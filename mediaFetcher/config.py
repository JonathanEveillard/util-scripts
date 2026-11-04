"""
This module intializes and pre-configures system for mediaFetcher's featured functionality.

It checks for any missing system dependencies required and installs them via brew.
"""

import sys
# import platform
import shutil
import subprocess
import os
from pathlib import Path

# Get OS type
# system = platform.system()


# Install dependency
def installDependency(package_name, cask=False):
    cmd = ["brew", "install"]
    if cask:
        cmd.append("--cask")
    cmd.append(package_name)

    result = subprocess.run(cmd, capture_output=True, text=True)

    if result.returncode == 0:
        print(f"Succesfully installed {package_name}")
        print(result.stdout)
    else:
        print(f"unsuccesful installation of {package_name}")
        print(result.stderr)

# Configure folder path
def createFolder():
    contentPath = "content"

    if not Path(contentPath).is_dir():
        print("Creating Content folder...")
        audioPath = os.path.join(contentPath, "audio")
        os.makedirs(audioPath, exist_ok=True)
        videoPath = os.path.join(contentPath, "video")
        os.makedirs(videoPath, exist_ok=True)
        print("Content folder created succesfully!")
        return
    
    subfolderExist = any(entry.is_dir() for entry in os.scandir(contentPath))

    if not subfolderExist: 
        print("Adding /audio /video to existing Content Folder")
        audioPath = os.path.join(contentPath, "audio")
        os.makedirs(audioPath, exist_ok=True)
        videoPath = os.path.join(contentPath, "video")
        os.makedirs(videoPath, exist_ok=True)
        return

    print("Content folder already exists and already contains subfolders!")

# Check dependencies
def checkDependency():
    if not shutil.which("brew"):
        install_brew = """NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)" """
        subprocess.run(install_brew, shell=True)
    activeDependencies = ["ffmpeg", "yt-dlp", "deno"]
    missingDependencies = []
    for dep in activeDependencies:
        if not shutil.which(dep):
            missingDependencies.append(dep)
    if missingDependencies:
        print("Missing Dependencies: ")
        for dep in missingDependencies:
            print(f"{dep:>2}")
        installDependencies = input("Install Missing Dependencies [y]/[n]: ")
        if installDependencies == "y":
            print("Installing Dependencies...")
            for dep in missingDependencies:
                installDependency(dep)
            print("All Dependencies have been Installed Successfully")
        elif installDependencies == 'n':
            print("Failed to install dependencies\nPlease Install required dependencies\nAborting...")
            sys.exit(0)
        else:
            checkDependency()
    else:
        print("All dependencies found.")
    return missingDependencies


def init():
    checkDependency()


