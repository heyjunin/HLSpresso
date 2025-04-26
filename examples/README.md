# HLSpresso Library Examples

This directory contains example Go programs demonstrating how to use the HLSpresso library (`pkg/transcoder`) for various video processing tasks.

## Running the Examples

1.  **Ensure FFmpeg/FFprobe are installed** and accessible in your system's PATH.
2.  **Navigate** into an example directory (e.g., `cd examples/local_to_hls`).
3.  **Get dependencies:** Run `go mod tidy` (you might need to initialize a module first with `go mod init example/<directory_name>` if running outside the main project structure for testing).
4.  **Place an input video:** Most examples expect an `input_video.mp4` file in the same directory. You can use any valid video file. The examples include code to create a small *dummy* file named `input_video.mp4` if it doesn't exist, allowing the code to run, but the resulting output won't be a valid video unless you replace it with a real one.
5.  **Run the example:** `go run main.go`.
6.  Check the console output and the generated output directory/file.

## Examples

*   `local_to_hls/`: Transcodes a local video file (`input_video.mp4`) to HLS format in the `output_hls_local` directory.
*   `remote_download_to_hls/`: Downloads a remote video (Big Buck Bunny sample) to `temp_downloads`, then transcodes it to HLS in `output_hls_remote_download`.
*   `remote_stream_to_hls/`: Transcodes a remote video (another sample) directly from the URL (using `StreamFromURL: true`) to HLS in `output_hls_remote_stream`.
*   `custom_resolutions/`: Transcodes a local video file to HLS using a predefined set of custom resolutions.
*   `to_mp4/`: Transcodes a local video file to a single MP4 file (`output_video.mp4`). 