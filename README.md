README
======

No frills OAI mirroring. It acts as cache and will take care of
retrieving new records.

Raw data will be stored in a hierarchy.

    ~/.oaimi/9c85aba8547acabeae024c51aa64ebc0322adc82/2015/08/10.xml
    ~/.oaimi/9c85aba8547acabeae024c51aa64ebc0322adc82/2015/08/11.xml

Installation
------------

    $ go get github.com/miku/oaimi/cmd/oaimi

Usage
-----

    $ oaimi http://www.example.com/oai/provider > metadata.xml

You can apply basic OAI filters:

    $ oaimi -set abc -from 2010-01-01 -until 2010-02-01 \
        http://www.example.com/oai/provider > metadata.xml
