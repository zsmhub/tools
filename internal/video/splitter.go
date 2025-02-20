/**
 * @description 视频切割工具包，提供视频切割和处理相关功能
 * @package video
 */
package video

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"tools/config"
)

/**
 * @description 视频切割器结构体
 */
type Splitter struct {
	cfg *config.Config
}

/**
 * @description 创建新的视频切割器实例
 * @return *Splitter 视频切割器实例
 */
func NewSplitter() *Splitter {
	return &Splitter{
		cfg: config.GetConfig(),
	}
}

/**
 * @description 获取视频总时长（秒）
 * @param videoPath 视频文件路径
 * @return duration 视频时长（秒）
 * @return error 错误信息
 */
func (s *Splitter) getVideoDuration(videoPath string) (float64, error) {
	cmd := exec.Command(s.cfg.FFmpeg.FFprobePath,
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
func (s *Splitter) getVideoResolution(videoPath string) (int, int, error) {
	cmd := exec.Command(s.cfg.FFmpeg.FFprobePath,
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
func (s *Splitter) needsResize(currentWidth, currentHeight int) bool {
	targetWidth := s.cfg.Video.Resolution.Width
	targetHeight := s.cfg.Video.Resolution.Height

	if targetWidth == 0 || targetHeight == 0 {
		return false
	}

	// 如果设置了强制调整尺寸，或者当前尺寸与目标尺寸不一致，就需要调整
	return s.cfg.Video.Resolution.ForceResize ||
		currentWidth != targetWidth ||
		currentHeight != targetHeight
}

/**
 * @description 检查视频是否包含字幕流
 * @param videoPath 视频文件路径
 * @return bool 是否包含字幕
 * @return error 错误信息
 */
func (s *Splitter) hasSubtitleStream(videoPath string) (bool, error) {
	cmd := exec.Command(s.cfg.FFmpeg.FFprobePath,
		"-v", "error",
		"-select_streams", "s",
		"-show_entries", "stream=codec_type",
		"-of", "csv=p=0",
		videoPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("检查字幕流失败: %v", err)
	}

	return strings.TrimSpace(string(output)) != "", nil
}

/**
 * @description 将视频切割成多个小片段
 * @param inputPath 输入视频路径
 * @param outputDir 输出目录
 * @return error 错误信息
 */
func (s *Splitter) Split(inputPath string, outputDir string) error {
	// 确保输出目录存在
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	// 获取视频时长
	totalDuration, err := s.getVideoDuration(inputPath)
	if err != nil {
		return err
	}

	// 计算实际的开始和结束时间
	startTime := s.cfg.Video.TimeRange.Start
	endTime := s.cfg.Video.TimeRange.End
	if endTime <= 0 || endTime > totalDuration {
		endTime = totalDuration
	}
	if startTime >= endTime {
		return fmt.Errorf("无效的时间范围：开始时间 %.2f 大于或等于结束时间 %.2f", startTime, endTime)
	}

	// 获取视频分辨率
	width, height, err := s.getVideoResolution(inputPath)
	if err != nil {
		return err
	}

	// 检查是否需要调整视频尺寸
	resizeNeeded := s.needsResize(width, height)

	// 检查是否包含字幕流
	hasSubtitle, err := s.hasSubtitleStream(inputPath)
	if err != nil {
		return err
	}

	// 如果视频没有字幕流，强制设置 Keep 为 false
	if !hasSubtitle {
		s.cfg.Video.Subtitle.Keep = false
	}

	// 计算需要切割的片段数
	duration := endTime - startTime
	segmentCount := int(math.Ceil(duration / float64(s.cfg.Video.MaxSegmentDuration)))

	// 获取输入文件的扩展名
	ext := filepath.Ext(inputPath)

	// 循环切割视频
	for i := 0; i < segmentCount; i++ {
		segmentStart := startTime + float64(i*s.cfg.Video.MaxSegmentDuration)
		segmentDuration := float64(s.cfg.Video.MaxSegmentDuration)

		// 处理最后一个片段的时长
		if segmentStart+segmentDuration > endTime {
			segmentDuration = endTime - segmentStart
			// 如果最后一个片段太短，就不处理
			if segmentDuration < 0.5 {
				break
			}
		}

		outputPath := filepath.Join(outputDir, fmt.Sprintf(s.cfg.Video.OutputNameFormat+"%s", i+1, ext))

		// 构建FFmpeg命令
		args := []string{
			"-i", inputPath,
			"-ss", fmt.Sprintf("%.3f", segmentStart),
			"-t", fmt.Sprintf("%.3f", segmentDuration),
			"-map", "0:v:0", // 明确指定视频流
			"-an", // 不包含音频
		}

		// 如果需要保留字幕且存在字幕流
		if s.cfg.Video.Subtitle.Keep && hasSubtitle {
			args = append(args,
				"-map", "0:s:0?", // 明确指定字幕流（如果存在）
				"-c:s", "copy", // 复制字幕流
			)
		} else {
			args = append(args, "-sn") // 不包含字幕
		}

		if resizeNeeded {
			// 添加视频尺寸调整参数
			args = append(args,
				"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2",
					s.cfg.Video.Resolution.Width,
					s.cfg.Video.Resolution.Height,
					s.cfg.Video.Resolution.Width,
					s.cfg.Video.Resolution.Height),
				"-c:v", "libx264", // 使用H.264编码器
				"-movflags", "+faststart", // 优化视频流媒体播放
				"-preset", s.cfg.Video.Quality.Preset, // 使用配置的编码速度预设
				"-crf", fmt.Sprintf("%d", s.cfg.Video.Quality.CRF), // 使用配置的视频质量参数
			)
		} else {
			// 不需要调整尺寸时的处理
			args = append(args,
				"-c:v", "copy", // 视频直接复制
			)
		}

		args = append(args,
			"-avoid_negative_ts", "make_zero", // 避免负时间戳
			"-max_muxing_queue_size", "1024", // 增加复用队列大小
			"-y", // 覆盖已存在的文件
			outputPath,
		)

		// 打印执行的命令（方便调试）
		fmt.Printf("执行命令: ffmpeg %s\n", strings.Join(args, " "))

		cmd := exec.Command(s.cfg.FFmpeg.FFmpegPath, args...)
		// 获取错误输出
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("切割视频片段 %d 失败: %v\nFFmpeg错误输出: %s", i+1, err, stderr.String())
		}
	}

	return nil
}
