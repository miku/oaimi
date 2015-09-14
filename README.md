README
======

No frills OAI mirroring. It acts as cache and will take care of
retrieving new records.

Installation
------------

    $ go get github.com/miku/oaimi/cmd/oaimi

Usage
-----

Simplest version:

    $ oaimi http://www.example.com/oai > metadata.xml

Apply OAI filters:

    $ oaimi -set abc -prefix marcxml -from 2010-01-01 -until 2010-02-01 \
        http://www.example.com/oai > metadata.xml

How it works
------------

The harvesting is splitted up into monthly chunks. The raw data is downloaded
and appended to a single file per source, set, prefix and month. Once a
month has been harvested successfully, the file is moved below a cache dir.

If you request the data for a given data source, `oaimi` will try to reuse the
cache and only go out to the interwebs to harvest not yet harvested parts. The
output file is the concatenated content for the requested date range.

The value proposition of `oaimi` is that you get a single file containing the
raw data for a specific source.

For the moment, any further processing must happen in the client (like
handling deletions).
