/**
 * @description 配置管理包，处理应用程序的所有配置相关操作
 * @package config
 */
package config

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config 配置结构体
type Config struct {
	FFmpeg FFmpegConfig `yaml:"ffmpeg"`
	Video  VideoConfig  `yaml:"video"`
}

// FFmpegConfig FFmpeg相关配置
type FFmpegConfig struct {
	FFmpegPath  string `yaml:"ffmpeg_path"`
	FFprobePath string `yaml:"ffprobe_path"`
}

// VideoConfig 视频处理相关配置
type VideoConfig struct {
	MaxSegmentDuration int              `yaml:"max_segment_duration"`
	OutputNameFormat   string           `yaml:"output_name_format"`
	Resolution         ResolutionConfig `yaml:"resolution"`
	// 视频质量配置
	Quality struct {
		// CRF值 (0-51, 越小质量越好)
		CRF int `yaml:"crf"`
		// 编码速度预设
		Preset string `yaml:"preset"`
	} `yaml:"quality"`
	// 字幕配置
	Subtitle struct {
		// 是否保留字幕
		Keep bool `yaml:"keep"`
		// 字幕编码（如 UTF-8）
		Encoding string `yaml:"encoding"`
	} `yaml:"subtitle"`
	// 视频处理时间范围
	TimeRange struct {
		// 开始时间（秒）
		Start float64 `yaml:"start"`
		// 结束时间（秒），0表示到视频结束
		End float64 `yaml:"end"`
	} `yaml:"time_range"`
}

// ResolutionConfig 视频分辨率配置
type ResolutionConfig struct {
	Width       int  `yaml:"width"`
	Height      int  `yaml:"height"`
	ForceResize bool `yaml:"force_resize"`
}

var (
	config *Config
	once   sync.Once
)

/**
 * @description 获取配置实例（单例模式）
 * @return *Config 配置实例
 */
func GetConfig() *Config {
	return config
}

/**
 * @description 加载配置文件
 * @param configPath 配置文件路径
 * @return error 错误信息
 */
func Load(configPath string) error {
	var err error
	once.Do(func() {
		config = &Config{}
		err = loadConfig(configPath)
	})
	return err
}

/**
 * @description 内部加载配置文件方法
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
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	// 设置默认值
	setDefaults()

	return nil
}

/**
 * @description 设置配置默认值
 */
func setDefaults() {
	if config.FFmpeg.FFmpegPath == "" {
		config.FFmpeg.FFmpegPath = "ffmpeg"
	}
	if config.FFmpeg.FFprobePath == "" {
		config.FFmpeg.FFprobePath = "ffprobe"
	}
	if config.Video.MaxSegmentDuration <= 0 {
		config.Video.MaxSegmentDuration = 10
	}
	if config.Video.OutputNameFormat == "" {
		config.Video.OutputNameFormat = "segment_%03d"
	}
	// 设置时间范围默认值
	if config.Video.TimeRange.Start < 0 {
		config.Video.TimeRange.Start = 0
	}
	// 设置字幕默认值
	if config.Video.Subtitle.Encoding == "" {
		config.Video.Subtitle.Encoding = "UTF-8"
	}
	if config.Video.Quality.CRF <= 0 || config.Video.Quality.CRF > 51 {
		config.Video.Quality.CRF = 18
	}
	if config.Video.Quality.Preset == "" {
		config.Video.Quality.Preset = "medium"
	}
}
