# tumblr-downloader

A tumblr scraper, designed to download all the images from the blogs that you want.

##Features

* **Iterative downloading** -- If you download from a blog, the run it again, only the missing images will be downloaded the second time.
* **Complete downloading** -- Will scan the entire blog for images, not just the first X pages.
* **Rate limiting**

## Download

Latest releases can be found [here](https://github.com/Liru/tumblr-downloader/releases/latest).

###Windows
[64 bit](https://github.com/Liru/tumblr-downloader/releases/download/v1.3.0/tumblr-downloader-windows.zip)

###Mac
[64 bit](https://github.com/Liru/tumblr-downloader/releases/download/v1.3.0/tumblr-downloader-mac.zip)

###Linux
[64 bit](https://github.com/Liru/tumblr-downloader/releases/download/v1.3.0/tumblr-downloader-linux.zip)

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

### Command line

Run `tumblr-downloader` as such, appending the usernames you want to download after the executable:

`$ ./tumblr-downloader nature-pics sunsets chickenpictures`

####Command line options

* `-d` - The maximum number of images to download at the same time. Default is 3.
* `-r` - The maximum number of requests per second to make.
* `-u` - Update mode -- the downloader will automatically stop once it reaches files that it has already downloaded.

## Contributing

1. Fork it!
2. Create your feature branch: `git checkout -b my-new-feature`
3. Commit your changes: `git commit -am 'Add some feature'`
4. Push to the branch: `git push origin my-new-feature`
5. Submit a pull request :D
