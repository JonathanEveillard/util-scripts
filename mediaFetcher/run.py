"""
mediaFetcher

Saves media for offline use, built on yt-dlp.
Checks that required tools like ffmpeg and Deno are installed first.
Intended for content you have permission to download.
"""

import config
import selection

def main():
    config.createFolder()
    selection.view()

if __name__ == "__main__":
    main();
