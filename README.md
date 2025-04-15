## sanny
A Go-based tool that downloads all files from a web directory to a specified local folder in batch.This downloader supports multi-threading and resumable downloads.

一个用 Go 语言编写的工具，用于从网络文件夹批量下载所有文件到本地指定目录，支持多线程和断点下载。

## usage
#exmaple
single thread(default)
.\sanny.exe -url "https://xfr139.larc.nasa.gov/sflops/Distribution/2025105001613_83698" -output "./my_downloads" 
mutithread
.\sanny.exe -url "https://xfr139.larc.nasa.gov/sflops/Distribution/2025105001613_83698" -output "./my_downloads" -thread 8

