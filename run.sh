#!/bin/bash

VIDEO_IN="$1"
VIDEO_OUT="$2"
HLS_TIME="$3"
FPS="$4"
GOP_SIZE="$5"
PRESET_P="$6"
out_dir="${10}"

PERCENTAGE="50" # Percentage where you want to extract the thumbnail, e.g., 50 for the midpoint

# Validate and adjust percentage
if [ "$PERCENTAGE" -gt 100 ]; then
    echo "Percentage is greater than 100, defaulting to 50"
    PERCENTAGE=50
elif [ "$PERCENTAGE" -lt 0 ]; then
    echo "Percentage cannot be negative, defaulting to 0"
    PERCENTAGE=50
fi

# Function to convert seconds to HH:MM:SS format
convert_seconds_to_time() {
    local total_seconds=$1
    printf "%02d:%02d:%02d\n" $(($total_seconds/3600)) $(($total_seconds%3600/60)) $(($total_seconds%60))
}

# Get the original video dimensions
DIMENSIONS=$(ffprobe -v error -select_streams v:0 -show_entries stream=width,height -of csv=p=0 "$VIDEO_IN")
if [ $? -ne 0 ]; then
    echo "Error executing ffprobe."
    exit 1
fi

# Read width and height into separate variables
ORIGINAL_WIDTH=$(echo "$DIMENSIONS" | cut -d ',' -f 1)
ORIGINAL_HEIGHT=$(echo "$DIMENSIONS" | cut -d ',' -f 2)

# Check if we successfully obtained width and height
if [ -z "$ORIGINAL_WIDTH" ] || [ -z "$ORIGINAL_HEIGHT" ]; then
    echo "Failed to retrieve video dimensions. ffprobe output: '$DIMENSIONS'"
    exit 1
fi

ASPECT_RATIO=$(echo "scale=6; $ORIGINAL_HEIGHT / $ORIGINAL_WIDTH" | bc)

# Define target resolutions
TARGET_HEIGHTS=(360 480 720) # 360p, 480p, 720p
TARGET_WIDTHS=()
BANDWIDTHS=("365k" "800k" "1500k")  # Bitrates for 360p, 480p, and 720p
MAX_RATES=("390k" "850k" "1600k")   # Max rates for 360p, 480p, and 720p
BUF_SIZES=("640k" "800k" "1500k")   # Buffer sizes for 360p, 480p, and 720p

# Calculate target WIDTHS based on aspect ratio and ensure they are even
for HEIGHT in "${TARGET_HEIGHTS[@]}"; do
    WIDTH=$(echo "scale=0; $HEIGHT / $ASPECT_RATIO" | bc)

    # Ensure WIDTH is even
    if [ $((WIDTH % 2)) -ne 0 ]; then
        WIDTH=$((WIDTH + 1))  # Round up to the nearest even number
    fi
    
    TARGET_WIDTHS+=("$WIDTH")
done

# Check if WIDTH were calculated correctly
for WIDTH in "${TARGET_WIDTHS[@]}"; do
    if [ -z "$WIDTH" ]; then
        echo "WIDTH calculation failed for one of the target Hights."
        exit 1
    fi
done

# Number of CPU cores
NUM_CORES=$(nproc)
echo ">>> Number of Cores: $NUM_CORES"

# HLS
ffmpeg -i "$VIDEO_IN" \
    -preset "$PRESET_P" -keyint_min "$GOP_SIZE" -g "$GOP_SIZE" -sc_threshold 0 -r "$FPS" -c:v libx264 -pix_fmt yuv420p \
    -map v:0 -s:v:0 "${TARGET_WIDTHS[0]}:${TARGET_HEIGHTS[0]}" -b:v:0 "${BANDWIDTHS[0]}" -maxrate:v:0 "${MAX_RATES[0]}" -bufsize:v:0 "${BUF_SIZES[0]}" \
    -map v:0 -s:v:1 "${TARGET_WIDTHS[1]}:${TARGET_HEIGHTS[1]}" -b:v:1 "${BANDWIDTHS[1]}" -maxrate:v:1 "${MAX_RATES[1]}" -bufsize:v:1 "${BUF_SIZES[1]}" \
    -map v:0 -s:v:2 "${TARGET_WIDTHS[2]}:${TARGET_HEIGHTS[2]}" -b:v:2 "${BANDWIDTHS[2]}" -maxrate:v:2 "${MAX_RATES[2]}" -bufsize:v:2 "${BUF_SIZES[2]}" \
    -map a:0 -map a:0 -map a:0 -c:a aac -b:a 128k -ac 1 -ar 44100 \
    -f hls -hls_time "$HLS_TIME" -hls_playlist_type vod -hls_flags independent_segments \
    -master_pl_name "$VIDEO_OUT.m3u8" \
    -hls_segment_filename "$out_dir/stream_%v/s%06d.ts" \
    -strftime_mkdir 1 \
    -var_stream_map "v:0,a:0 v:1,a:1 v:2,a:2" "$out_dir/stream_%v.m3u8" \
    -threads "$NUM_CORES" -y # Multithreading: Ensure FFmpeg uses all available cores

# Check if the transcoding was successful
if [ $? -eq 0 ]; then
    # Remove out_dir text from all .m3u8 files which is not needed and causes issues while playing video for not removed
    for m3u8_file in $(find "$out_dir" -type f -name "*.m3u8"); do
        sed -i "s|$out_dir/||g" "$m3u8_file"
    done

    # Get the video duration in seconds for thumbnail
    DURATION=$(ffprobe -v error -select_streams v:0 -show_entries stream=duration -of csv=p=0 "$VIDEO_IN")
    DURATION_SEC=$(printf "%.0f" "$DURATION")

    # Calculate the timestamp based on the percentage for thumbnail
    TIMESTAMP_SEC=$(printf "%.0f" $(echo "$DURATION_SEC * $PERCENTAGE / 100" | bc))
    TIMESTAMP=$(convert_seconds_to_time "$TIMESTAMP_SEC")

    echo "Percent thumbnail frame: $PERCENTAGE"
    echo "Video Duration: $DURATION_SEC seconds"
    echo "Extracting thumbnail at: $TIMESTAMP ($TIMESTAMP_SEC seconds)"

    # Generate a thumbnail from the video
    ffmpeg -i "$VIDEO_IN" -ss "$TIMESTAMP" -vframes 1 "$out_dir/thumbnail.jpg" -y
else
    echo "Transcoding failed. Thumbnail will not be created."
fi