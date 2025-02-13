# drone-robot

## Building

Build the plugin binary:

```text
scripts/build.sh
```

Build the plugin image:

```text
docker build -t plugins/robot -f docker/Dockerfile .
```

## Testing

Execute the plugin from your current working directory:
## This plugin processes Robot Framework XML report files (output.xml) and logs the test results in the console and also write stats to DRONE_OUTPUT evn variable.
- It supports various configurations for handling critical, skipped, and failed tests, and enforces thresholds for stopping the build based on the number of failures.

```
docker run --rm \
  -e PLUGIN_REPORT_DIRECTORY="./reports" \
  -e PLUGIN_REPORT_FILE_NAME_PATTERN="output.xml" \
  -e PLUGIN_PASS_THRESHOLD=90 \
  -e PLUGIN_UNSTABLE_THRESHOLD=80 \
  -e PLUGIN_COUNT_SKIPPED_TESTS=true \
  -e PLUGIN_ONLY_CRITICAL=false \
  -e PLUGIN_LOG_LEVEL="info" \
  -v $(pwd):$(pwd) \
  plugins/robot
```
## Example Harness Step:
```
- step:
    identifier: robot-report-processing
    name: Robot Framework Report Processing
    spec:
      image: plugins/robot
      settings:
        report_directory: "./reports"
        report_file_name_pattern: "output.xml"
        pass_threshold: 90
        unstable_threshold: 80
        count_skipped_tests: true
        only_critical: false
        level: "info"
    timeout: ''
    type: Plugin
```

## Plugin Settings
- `PLUGIN_REPORT_DIRECTORY`
Description: The directory where output.xml reports are located.
Example: ./reports

- `PLUGIN_REPORT_FILE_NAME_PATTERN`
Description: The Robot Framework report file name.
Example: output.xml

- `PLUGIN_COUNT_SKIPPED_TESTS`
Description: This flag determines whether skipped tests should be counted in the final test statistics.
Example: true

- `PLUGIN_ONLY_CRITICAL`
Description: This flag ensures that only critical tests (tests marked with critical="yes") are considered in the statistics.
Example: false

- `PLUGIN_PASS_THRESHOLD`
Description: The number of passed tests required for the build to be marked as successful.
Example: 90

- `PLUGIN_UNSTABLE_THRESHOLD`
Description: The number of passed tests below which the build is marked as unstable.
Example: 80
	
- `PLUGIN_LOG_LEVEL`
Description: Defines the plugin log level. Set to debug for detailed logs.
Example: info