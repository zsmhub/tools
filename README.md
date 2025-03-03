# tools

写一些辅助脚本

## 项目结构

- cmd/ 存放主程序入口
- config/ 处理所有配置相关逻辑
- internal/ 存放内部包

## 使用方法

```bash
# 视频分割
go run cmd/video_splitter/main.go <输入视频路径> <输出目录>

# 视频格式转换
go run cmd/video_transfer/main.go <输入视频路径> <输出目录> <格式>
```
