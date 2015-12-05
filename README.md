README
======

> The Open Archives Initiative Protocol for Metadata Harvesting (OAI-PMH) is a low-barrier mechanism for repository interoperability. https://www.openarchives.org/pmh/

No frills OAI harvesting. It acts as cache and will take care of incrementally retrieving new records.

[![Build Status](https://travis-ci.org/miku/oaimi.svg?branch=master)](https://travis-ci.org/miku/oaimi)

![](https://github.com/miku/oaimi/blob/master/img/convergent_35855_sm.gif)

Installation
------------

    $ go get github.com/miku/oaimi/cmd/oaimi

There are [deb and rpm packages](https://github.com/miku/oaimi/releases) as well.

Usage
-----

Show repository information:

    $ oaimi -id http://digital.ub.uni-duesseldorf.de/oai
    {
      "formats": [
        {
          "prefix": "oai_dc",
          "schema": "http://www.openarchives.org/OAI/2.0/oai_dc.xsd"
        },
        ...
        {
          "prefix": "epicur",
          "schema": "http://www.persistent-identifier.de/xepicur/version1.0/xepicur.xsd"
        }
      ],
      "identify": {
        "name": "Visual Library Server der Universitäts- und Landesbibliothek Düsseldorf",
        "url": "http://digital.ub.uni-duesseldorf.de/oai/",
        "version": "2.0",
        "email": "docserv@uni-duesseldorf.de",
        "earliest": "2008-04-18T07:54:14Z",
        "delete": "no",
        "granularity": "YYYY-MM-DDThh:mm:ssZ"
      },
      "sets": [
        {
          "spec": "ulbdvester",
          "name": "Sammlung Vester (DFG)"
        },
        ...
        {
          "spec": "ulbd_rsh",
          "name": "RSH"
        }
      ]
    }

Harvest the complete repository into a single file (default format is [oai_dc](http://www.openarchives.org/OAI/2.0/oai_dc.xsd), might take a few minutes on first run):

    $ oaimi -verbose http://digital.ub.uni-duesseldorf.de/oai > metadata.xml

Harvest only a slice (e.g. set *ulbdvester* in format *epicur* for *2010* only):

    $ oaimi -set ulbdvester -prefix epicur -from 2010-01-01 \
            -until 2010-12-31 http://digital.ub.uni-duesseldorf.de/oai > slice.xml

Harvest, and add an artificial root element, so the result gets a bit more valid XML:

    $ oaimi -root records http://digital.ub.uni-duesseldorf.de/oai > withroot.xml

To list the harvested files, run:

    $ ls $(oaimi -dirname http://digital.ub.uni-duesseldorf.de/oai)

Add any parameter to see the resulting cache dir:

    $ ls $(oaimi -dirname -set ulbdvester -prefix epicur -from 2010-01-01 \
                 -until 2010-12-31 http://digital.ub.uni-duesseldorf.de/oai)

To remove all cached files:

    $ rm -rf $(oaimi -dirname http://digital.ub.uni-duesseldorf.de/oai)

Options:

    $ oaimi -h
    Usage of oaimi:
      -cache string
          oaimi cache dir (default "/Users/tir/.oaimicache")
      -dirname
          show shard directory for request
      -from string
          OAI from
      -id
          show repository info
      -prefix string
          OAI metadataPrefix (default "oai_dc")
      -root string
          name of artificial root element tag to use
      -set string
          OAI set
      -until string
          OAI until (default "2015-11-30")
      -v  prints current program version
      -verbose
          more output

Experimental `oaimi-id` and `oaimi-sync` for identifying or harvesting in parallel:

    $ oaimi-id -h
    Usage of oaimi-id:
      -timeout duration
          deadline for requests (default 30m0s)
      -v  prints current program version
      -verbose
          be verbose
      -w int
          requests in parallel (default 8)

    $ oaimi-sync
    Usage of oaimi-sync:
      -cache string
          where to cache responses (default "/Users/tir/.oaimicache")
      -v  prints current program version
      -verbose
          be verbose
      -w int
          requests in parallel (default 8)

How it works
------------

The harvesting is performed in chunks (weekly at the moment). The raw data is
downloaded and appended to a single temporary file per source, set, prefix and
month. Once a month has been harvested successfully, the temporary file is
moved below a cache dir. In short: The cache dir will not contain partial files.

If you request the data for a given data source, `oaimi` will try to reuse the
cache and only harvest not yet cached data. The output file is the
concatenated content for the requested date range. The output is no valid XML
because a root element is missing. You can add a custom root element with the
`-root` flag.

The value proposition of `oaimi` is that you get a single file containing the
raw data for a specific source with a single command and that incremental
updates are relatively cheap - at most the last 7 days need to be fetched.

For the moment, any further processing must happen in the client (like
handling deletions).

More Docs: http://godoc.org/github.com/miku/oaimi

Similar projects
----------------

* [oai-harvest-manager](https://github.com/TheLanguageArchive/oai-harvest-manager)
* [Catmandu::OAI](https://github.com/LibreCat/Catmandu-OAI)
* [Sickle](https://pypi.python.org/pypi/Sickle)

More sites
----------

* http://roar.eprints.org/listfriends.xml
* http://www.openarchives.org/pmh/registry/ListFriends
* http://gita.grainger.uiuc.edu/registry/ListAllAllRepos.asp?format=xml
* https://centres.clarin.eu/oai_pmh
* [config-others.xml](https://github.com/TheLanguageArchive/oai-harvest-manager/blob/a4ee9e72c0162a664e1b0ebd71b36b3f2f4eea71/src/main/resources/config-others.xml#L75)

Distributions
-------------

Over 2038 repositories.

* supported [formats](https://gist.github.com/anonymous/92ec3e297963b98c0bc7)
* [earliest](https://gist.github.com/anonymous/37e01bb984f9ce6fd3ec) date
* Format [representants](https://gist.github.com/miku/3679c3dc298796d38d2d)

Miscellaneous
-------------

* [1min](https://asciinema.org/a/6pkf42xpx6mpcwzupffo0iz1d?autoplay=1) of harvest, 2min [parallelism](https://asciinema.org/a/ce9g796vdxb9dk3g1qhtx0q8v?autoplay=1)

License
-------

* GPLv3
* This project uses [ioutil2](https://github.com/youtube/vitess/blob/c0366e645cb76048c4b30dbeffd8dc686697eb6f/go/ioutil2/ioutil.go), Copyright 2012, Google Inc. All rights reserved.
  Use of this source code is governed by a BSD-style license.
