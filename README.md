[![Build Status](https://travis-ci.org/hauke96/pico.svg?branch=master)](https://travis-ci.org/hauke96/pico)
# pico
PICO is a mechanism to compress images using interpolated data. This repo contains a converter and viewer for PICO.
# Screenshot 
This image shows a photo compressed with an accuracy tolerance (s.blow) of 2.
![Image or link broken :( please report](http://hauke-stieler.de/public/pico/pico_screenshot_01.png "PICO version v0.1")
# How it works
TODO
## The algorithm
## The format
# Problems
* The quality is pretty bad and the size of the file is not that small. By choosing an accuracy tolerance of 2.5 or lower the ipf file might be greater then an lossless compressed PNG file which is pretty bad.
* The spereate compression of the color channels does not work well near color gradients.
