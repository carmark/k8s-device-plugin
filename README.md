[![BK Pipelines Status](https://api.bkdevops.qq.com/process/api/external/pipelines/projects/mock-k8s-device-plugin/p-5e347ba04b48421585a492dfde05453e/badge?X-DEVOPS-PROJECT-ID=mock-k8s-device-plugin)](http://api.devops.oa.com/process/api-html/user/builds/projects/mock-k8s-device-plugin/pipelines/p-5e347ba04b48421585a492dfde05453e/latestFinished?X-DEVOPS-PROJECT-ID=mock-k8s-device-plugin)

# Mock kubernetes device plugin 

```
$ ./k8s-device-plugin --help
Usage of ./k8s-device-plugin:
  -add_dir_header
    	If true, adds the file directory to the header
  -alsologtostderr
    	log to standard error as well as files
  -log_backtrace_at value
    	when logging hits line file:N, emit a stack trace
  -log_dir string
    	If non-empty, write log files in this directory
  -log_file string
    	If non-empty, use this log file
  -log_file_max_size uint
    	Defines the maximum size a log file can grow to. Unit is megabytes. If the value is 0, the maximum file size is unlimited. (default 1800)
  -logtostderr
    	log to standard error instead of files (default true)
  -resource-name string
    	Define the default resource name. (default "nvidia.com/gpu")
  -resource-num int
    	Define the default resource number. (default 8)
  -skip_headers
    	If true, avoid header prefixes in the log messages
  -skip_log_headers
    	If true, avoid headers when opening log files
  -stderrthreshold value
    	logs at or above this threshold go to stderr (default 2)
  -v value
    	number for the log level verbosity
  -vmodule value
    	comma-separated list of pattern=N settings for file-filtered logging
```
