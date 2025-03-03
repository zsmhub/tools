package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
	"tools/config"

	"github.com/google/uuid"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Println("使用方法: video_splitter <输入视频路径> <输出目录> <格式>")
		os.Exit(1)
	}

	inputPath := os.Args[1]
	outputDir := os.Args[2]
	formatExt := os.Args[3]

	// 加载配置文件
	configPath := "config/config.yaml"
	if err := config.Load(configPath); err != nil {
		fmt.Printf("加载配置文件失败: %v\n", err)
		os.Exit(1)
	}

	outputFile := outputDir + "/" + uuid.New().String() + "." + formatExt

	// 创建带超时的context - 增加到15分钟
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	cfg := config.GetConfig()

	// 构建ffmpeg命令，优化转换参数
	ffmpegCmd := fmt.Sprintf("%s -i %s -c:v libx264 -preset veryfast "+
		// 视频编码参数优化
		"-tune film "+ // 针对电影内容优化
		"-profile:v high "+ // 使用高规格配置
		"-level:v 4.0 "+ // 兼容性设置
		"-crf 23 "+ // 平衡质量和大小的压缩率
		"-maxrate 4M -bufsize 8M "+ // 控制码率
		"-g 50 "+ // 关键帧间隔
		"-keyint_min 25 "+ // 最小关键帧间隔
		"-sc_threshold 40 "+ // 场景切换阈值
		"-threads 0 "+ // 自动选择线程数
		// 音频编码参数
		"-c:a aac -b:a 128k "+
		// 其他参数
		"-movflags +faststart "+ // 支持快速播放
		"-y %s", // 覆盖输出文件
		cfg.FFmpeg.FFmpegPath,
		inputPath,
		outputFile,
	)

	// 使用CommandContext替代Command来支持超时控制
	cmd := exec.CommandContext(ctx, cfg.FFmpeg.BashPath, "-c", ffmpegCmd)

	// 执行命令并获取输出
	out, err := cmd.CombinedOutput()
	if err != nil {
		// 检查是否是超时导致的错误
		if ctx.Err() == context.DeadlineExceeded {
			// 如果是超时错误，清理临时文件
			_ = os.Remove(outputFile)
			fmt.Println("media format conversion timeout after 15 minutes")
		}
		// 其他错误
		_ = os.Remove(outputFile)
		fmt.Println("cmd err: " + err.Error() + "->" + string(out))
	}

	fmt.Println("media format conversion success")
}
