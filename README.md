README
======

> The Open Archives Initiative Protocol for Metadata Harvesting (OAI-PMH) is a low-barrier mechanism for repository interoperability.

No frills OAI harvesting and mirroring. It acts as cache and will take care of
retrieving new records.

![](https://github.com/miku/oaimi/blob/master/img/convergent_35855_sm.gif)

Installation
------------

    $ go get github.com/miku/oaimi/cmd/oaimi

There are [deb and rpm packages](https://github.com/miku/oaimi/releases) as well.

Usage
-----

Simplest version:

    $ oaimi http://www.example.com/oai > metadata.xml

Apply OAI filters:

    $ oaimi -set abc -prefix marcxml -from 2010-01-01 -until 2010-02-01 \
        http://www.example.com/oai > metadata.xml

Example:

    $ oaimi -verbose http://digital.ub.uni-duesseldorf.de/oai > metadata.xml

To list the files, run:

    $ ls $(oaimi -dirname http://digital.ub.uni-duesseldorf.de/oai)

To empty all cached files:

    $ rm -rf $(oaimi -dirname http://digital.ub.uni-duesseldorf.de/oai)

Options:

    $ oaimi -h
    Usage of oaimi:
      -cache string
          oaimi cache dir (default "/Users/tir/.oaimi")
      -dirname
          show shard directory for request
      -from string
          OAI from
      -from-earliest
          harvest from earliest timestamp
      -id
          show repository information
      -prefix string
          OAI metadataPrefix (default "oai_dc")
      -retry int
          retry count for exponential backoff (default 10)
      -root string
          name of artificial root element tag to use
      -set string
          OAI set
      -until string
          OAI until (default "2015-10-31")
      -v  prints current program version
      -verbose
          more output

How it works
------------

The harvesting is performed in monthly chunks. The raw data is downloaded and
appended to a single temporary file per source, set, prefix and month. Once a
month has been harvested successfully, the temporary file is moved below a
cache dir. In short: The cache dir will not contain half-downloaded files.

If you request the data for a given data source, `oaimi` will try to reuse the
cache and only harvest not yet cached data. The output file is the
concatenated content for the requested date range. The output is no valid XML
because a root element is missing. You can add a custom root element with the
`-root` flag.

The value proposition of `oaimi` is that you get a single file containing the
raw data for a specific source with a single command and that incremental
updates are relatively cheap - at most the last 30 days need to be fetched.

For the moment, any further processing must happen in the client (like
handling deletions).

More Docs: http://godoc.org/github.com/miku/oaimi

Similar projects
----------------

* [Catmandu::OAI](https://github.com/LibreCat/Catmandu-OAI)
* [Sickle](https://pypi.python.org/pypi/Sickle)

TODO
----

* use some kind of decorator pattern: `Batch{Cached{Request}}`
