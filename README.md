README
======

WIP.

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

Query for the number of documents:

    $ oaimi -size http://www.example.com/oai/provider
    29871

How it works
------------

The harvesting is splitted up into daily chunks. The raw data is downloaded
and appended to a single file per source, set, prefix and day. Once a day has
been harvested successfully, the file is moved below a cache dir.

If you request the data for a given data source, `oaimi` will try to reuse the
cache and only harvest not yet harvested dates. The output file is the
concatenated content for the requested date range.

The value of `oaimi` is that you get a single file containing the raw data for
a specific source. Any further processing must happen in the client.
