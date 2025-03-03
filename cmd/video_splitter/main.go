/**
 * @description 视频切割工具主程序
 */
package main

import (
	"fmt"
	"os"

	"tools/config"
	"tools/internal/video"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("使用方法: video_splitter <输入视频路径> <输出目录>")
		os.Exit(1)
	}

	inputPath := os.Args[1]
	outputDir := os.Args[2]

	// 加载配置文件
	configPath := "config/config.yaml"
	if err := config.Load(configPath); err != nil {
		fmt.Printf("加载配置文件失败: %v\n", err)
		os.Exit(1)
	}

	// 创建视频切割器实例
	splitter := video.NewSplitter()

	// 执行视频切割
	if err := splitter.Split(inputPath, outputDir); err != nil {
		fmt.Printf("视频切割失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("视频切割完成！")
}
