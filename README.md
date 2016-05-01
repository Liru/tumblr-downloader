# tumblr-downloader 

A tumblr scraper, designed to download all the images from the blogs that you want.

[![Go Report Card](https://goreportcard.com/badge/github.com/liru/tumblr-downloader)](https://goreportcard.com/report/github.com/liru/tumblr-downloader)
[![MIT licensed](https://img.shields.io/github/license/liru/tumblr-downloader.svg)](https://raw.githubusercontent.com/liru/tumblr-downloader/master/LICENSE)
[![Version](https://img.shields.io/github/release/liru/tumblr-downloader.svg)
[![Downloads](https://img.shields.io/github/downloads/liru/tumblr-downloader/total.svg)](https://github.com/liru/tumblr-downloader/releases)

##Features

* **Photo, video, and audio downloading**
* **Iterative downloading** -- If you download from a blog, the run it again, only the missing downloadables can be downloaded the second time.
* **Complete downloading** -- Will scan the entire blog for downloadables, not just the first X pages.
* **Rate limiting**
* **Concurrency** -- download from multiple blogs at the same time
* **GfyCat support** -- download linked WebM and MP4 files from GfyCat 

## Download

Latest releases can be found [here](https://github.com/Liru/tumblr-downloader/releases/latest) for Windows, Mac, and Linux.

## Usage
###Simple
Make a text file called `download.txt` with each tumblr blog you want to download on a separate line:
```
nature-pics
sunsets
chickenpictures
```

Run `tumblr-downloader` once it's complete.  It'll download all the pictures from the blog and save it in a `downloads/<username>` folder for each user.

You can also download a single tag for a blog, if you only want specific content. For example, you can have the following:
```
nature-pics forests
sunsets
chickenpictures funny faces
```

If your tag has spaces in it, just type the tag normally after the blog name. For instance, in the above example, `chickenpictures` will download anything tagged with `funny faces`. (Note that it will NOT download `funny` and `faces` separately like this.)

#### Command line options

* `-f` - Force check -- the downloader will recheck old tumblr posts to see if it missed anything.
* `-ignore-audio`, `-ignore-videos`, `-ignore-photos` - Skips downloading the respective types of files.
* `-p` - Enable progress bar to track progress instead of printing files being downloaded.

## Suggestions

Use the `issues` tab provided by Github at the top of this project's page.

## Contributing

1. Fork it!
2. Create your feature branch: `git checkout -b my-new-feature`
3. Commit your changes: `git commit -am 'Add some feature'`
4. Push to the branch: `git push origin my-new-feature`
5. Submit a pull request :D
