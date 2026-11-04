# mediaFetcher

Darwin based tool to retrieve content from the internet through CLI

![gif](./gif.gif)

## How to use

- run `python run.py` to preconfigure system and install contents

## How it works
- `init.py` **DEPRECATED** (To be deleted)
- `config.py` 
    - Downloads any missing system dependencies required (ffmpeg, brew, ...)
    - Configure file path for script functionality
- `selection.py`
    - Creates CLI interaction for installation method type
- `run.py` 
    - Runs `config.py` `selection.py` in sequential and logical order


