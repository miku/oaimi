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
      -timeout duration
          request timeout (default 1m0s)
      -until string
          OAI until (default "2015-11-02")
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

* [oai-harvest-manager](https://github.com/TheLanguageArchive/oai-harvest-manager)
* [Catmandu::OAI](https://github.com/LibreCat/Catmandu-OAI)
* [Sickle](https://pypi.python.org/pypi/Sickle)

More sites
==========

* http://roar.eprints.org/listfriends.xml
* http://www.openarchives.org/pmh/registry/ListFriends
* http://gita.grainger.uiuc.edu/registry/ListAllAllRepos.asp?format=xml
* https://centres.clarin.eu/oai_pmh
* [config-others.xml](https://github.com/TheLanguageArchive/oai-harvest-manager/blob/a4ee9e72c0162a664e1b0ebd71b36b3f2f4eea71/src/main/resources/config-others.xml#L75)

Format distribution
-------------------

Format distribution over 2038 repositories: https://gist.github.com/anonymous/92ec3e297963b98c0bc7.
