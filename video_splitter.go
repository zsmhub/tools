/**
 * @description 视频切割工具，将大视频文件切割成多个小视频片段
 * @method SplitVideo - 主要切割方法
 * @method getVideoDuration - 获取视频时长
 */

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 配置结构体
type Config struct {
	FFmpeg struct {
		FFmpegPath  string `yaml:"ffmpeg_path"`
		FFprobePath string `yaml:"ffprobe_path"`
	} `yaml:"ffmpeg"`
	Video struct {
		MaxSegmentDuration int    `yaml:"max_segment_duration"`
		OutputNameFormat   string `yaml:"output_name_format"`
		Resolution         struct {
			Width       int  `yaml:"width"`
			Height      int  `yaml:"height"`
			ForceResize bool `yaml:"force_resize"`
		} `yaml:"resolution"`
	} `yaml:"video"`
}

// 全局配置变量
var config Config

/**
 * @description 加载配置文件
 * @param configPath 配置文件路径
 * @return error 错误信息
 */
func loadConfig(configPath string) error {
	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 解析YAML
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	// 验证必要的配置
	if config.FFmpeg.FFmpegPath == "" {
		config.FFmpeg.FFmpegPath = "ffmpeg" // 默认使用PATH中的ffmpeg
	}
	if config.FFmpeg.FFprobePath == "" {
		config.FFmpeg.FFprobePath = "ffprobe" // 默认使用PATH中的ffprobe
	}
	if config.Video.MaxSegmentDuration <= 0 {
		config.Video.MaxSegmentDuration = 10 // 默认10秒
	}
	if config.Video.OutputNameFormat == "" {
		config.Video.OutputNameFormat = "segment_%03d" // 默认输出文件名格式
	}

	return nil
}

/**
 * @description 获取视频总时长（秒）
 * @param videoPath 视频文件路径
 * @return duration 视频时长（秒）
 * @return error 错误信息
 */
func getVideoDuration(videoPath string) (float64, error) {
	// 使用ffprobe获取视频时长
	cmd := exec.Command(config.FFmpeg.FFprobePath,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("获取视频时长失败: %v", err)
	}

	duration, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, fmt.Errorf("解析视频时长失败: %v", err)
	}

	return duration, nil
}

/**
 * @description 获取视频分辨率
 * @param videoPath 视频文件路径
 * @return width 视频宽度
 * @return height 视频高度
 * @return error 错误信息
 */
func getVideoResolution(videoPath string) (int, int, error) {
	cmd := exec.Command(config.FFmpeg.FFprobePath,
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=s=x:p=0",
		videoPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("获取视频分辨率失败: %v", err)
	}

	// 解析输出 "1920x1080"
	parts := strings.Split(strings.TrimSpace(string(output)), "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("解析视频分辨率失败")
	}

	width, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("解析视频宽度失败: %v", err)
	}

	height, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("解析视频高度失败: %v", err)
	}

	return width, height, nil
}

/**
 * @description 检查是否需要调整视频尺寸
 * @param currentWidth 当前视频宽度
 * @param currentHeight 当前视频高度
 * @return bool 是否需要调整尺寸
 */
func needsResize(currentWidth, currentHeight int) bool {
	targetWidth := config.Video.Resolution.Width
	targetHeight := config.Video.Resolution.Height

	if targetWidth == 0 || targetHeight == 0 {
		return false
	}

	if config.Video.Resolution.ForceResize {
		return currentWidth != targetWidth || currentHeight != targetHeight
	}

	return currentWidth > targetWidth || currentHeight > targetHeight
}

/**
 * @description 将视频切割成多个小片段
 * @param inputPath 输入视频路径
 * @param outputDir 输出目录
 * @return error 错误信息
 */
func SplitVideo(inputPath string, outputDir string) error {
	// 确保输出目录存在
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	// 获取视频时长
	duration, err := getVideoDuration(inputPath)
	if err != nil {
		return err
	}

	// 计算需要切割的片段数
	segmentCount := int(duration/float64(config.Video.MaxSegmentDuration)) + 1

	// 获取输入文件的扩展名
	ext := filepath.Ext(inputPath)

	// 获取视频分辨率
	width, height, err := getVideoResolution(inputPath)
	if err != nil {
		return err
	}

	// 检查是否需要调整视频尺寸
	resizeNeeded := needsResize(width, height)

	// 循环切割视频
	for i := 0; i < segmentCount; i++ {
		startTime := i * config.Video.MaxSegmentDuration
		outputPath := filepath.Join(outputDir, fmt.Sprintf(config.Video.OutputNameFormat+"%s", i+1, ext))

		// 构建FFmpeg命令
		args := []string{
			"-i", inputPath,
			"-ss", fmt.Sprintf("%d", startTime),
			"-t", fmt.Sprintf("%d", config.Video.MaxSegmentDuration),
		}

		if resizeNeeded {
			// 添加视频尺寸调整参数
			args = append(args,
				"-vf", fmt.Sprintf("scale=%d:%d",
					config.Video.Resolution.Width,
					config.Video.Resolution.Height),
				"-c:v", "libx264", // 使用H.264编码器
				"-preset", "medium", // 编码速度和质量的平衡
				"-crf", "23", // 视频质量参数，范围0-51，越小质量越好
				"-c:a", "copy", // 音频直接复制
			)
		} else {
			args = append(args, "-c", "copy") // 不需要调整尺寸时直接复制
		}

		args = append(args,
			"-y", // 覆盖已存在的文件
			outputPath,
		)

		cmd := exec.Command(config.FFmpeg.FFmpegPath, args...)

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("切割视频片段 %d 失败: %v", i+1, err)
		}
	}

	return nil
}

func main() {
	if len(os.Args) != 3 && len(os.Args) != 4 {
		fmt.Println("使用方法: video_splitter <输入视频路径> <输出目录> [配置文件路径]")
		os.Exit(1)
	}

	inputPath := os.Args[1]
	outputDir := os.Args[2]

	// 加载配置文件
	configPath := "config.yaml"
	if len(os.Args) == 4 {
		configPath = os.Args[3]
	}

	if err := loadConfig(configPath); err != nil {
		fmt.Printf("加载配置文件失败: %v\n", err)
		os.Exit(1)
	}

	if err := SplitVideo(inputPath, outputDir); err != nil {
		fmt.Printf("视频切割失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("视频切割完成！")
}
