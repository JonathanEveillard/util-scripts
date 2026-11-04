import os
import subprocess

audioPath = "./content/audio/"
videoPath = "./content/video/"

def getURL():
    inputUrl = input("Enter your url or 'q' to exit: ")
    url = str(inputUrl)
    return url


def fetchVideo():
    url = getURL()
    if url.lower() == 'q':
        return

    print("Fetching audio, URL defined as: " + url)
    cmd = [
        "yt-dlp",
        "--no-playlist",
        "-f", "bv*[vcodec^=avc1]+ba[ext=m4a]/b[ext=mp4]",
        "--merge-output-format", "mp4",
        url
    ]
    result = subprocess.run(
                cmd, 
                cwd = videoPath,
                capture_output= False,
                text = True,
            )
    if result.returncode == 0:
        print("Video Download succesful!")



def fetchAudio():
    url = getURL()
    if url.lower() == 'q':
        return
        
    print("Fetching Audio, URL defined as: " + url)
    
    cmd = [
        "yt-dlp",
        "--no-playlist",
        "-x",
        "--audio-format", "m4a",
        url
    ]
    
    result = subprocess.run(
                cmd,
                cwd=audioPath,
                capture_output=False,
                text=True,
            )
    if result.returncode == 0:
        print("Audio Download successful!")

def view():
    print("[0]: Audio\n[1]: Video")
    userInput = input("Select your installation type: ")
    if userInput == "0":
        fetchAudio()
    elif userInput == "1":
        fetchVideo()
    else:
        print("Invalid choice! Please choose 0 or 1")
