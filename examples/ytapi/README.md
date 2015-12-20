
ytapi
=====

Command Line Tool for calling the YouTube API's
(c) Copyright David Thorpe 2015 All Rights Reserved
Please see LICENSE file for usage information

This command-line tool provides some basic operations for manipulating and getting
information for YouTube using the rich set of YouTube API's. Different operations
will be implemented over time. Authentication is either through OAuth or Service
Accounts, if you authenticate against a YouTube Content Owner.

Building the tool
-----------------

In order to build the tool into a binary, use the following command, once you
have the Go package installed and have set your `$GOBIN` environment variable:

```
bash% cd $GOPATH/src/github/djthorpe/gopi
bash% go install examples/ytapi/ytapi.go 
bash% ls -l $GOBIN/ytapi
```

Authentication Set-up
---------------------

For OAuth authentication (where you authenticate against your own YouTube channel)
you'll need to create a Google Developer project and enable it for the YouTube API,
then create a client secret JSON file, which should be stored in the "~/.credentials"
folder named "client_secret.json". This ensures that API calls are made against
your own quota.

For Service Account authentication (where you authenticate against a YouTube
Content Owner) you'll need to create service account credentials, add the
Service Account Email address to your content owner, and download the service
account JSON file, which should be stored in the "~/.credentials" folder names
"service_account.json". Within the developer console, ensure you enable access
to both the YouTube API and the YouTube Partner API.

Generic Flags
-------------

The command line tool has a number of generic flags which can be used for many
of the operations:

  * `--contentowner <id>` Set the content owner used to authenticate your
    service account information to.
  * `--debug` Show HTTP traffic on the command line when making the calls. Shows
    both the requests and the responses
  * `--authtoken <filename>` Name of the cached authentication token, in case
    you want to authenticate to several different YouTube accounts simulatenously

There are several other generic flags which you can see if you use the `--help` flag
when running the tool.

Getting information
-------------------

The commands for getting information from YouTube are as follows:

  * `ytapi channels` Will display a list of channels which can be controlled. In
	general, you'll get a single channel back if using OAuth authentication, or
	several channels when using service account authentication.
  * `ytapi (--channel <id>) videos` Will list video uploads for all channels, or
    a single channel when using the channel flag.
  * `ytapi (--channel <id> --status <statusfilter> broadcasts` will list live
    stream broadcasts in one or more channels.
  * `ytapi (--channel <id>) streams` will list streams in one or more channels.

Live Stream Control
-------------------

You can control live streams using the following commands:

  * `ytapi --stream <key> --video <id> bind` Will bind an incoming stream to a 
    live broadcast.
  * `ytapi --video <id> unbind` Will disassociate a stream from a live broadcast.

To Do
-----

This command-line tool is still in development, clearly there are a lot of API
calls missing at this time.




  
