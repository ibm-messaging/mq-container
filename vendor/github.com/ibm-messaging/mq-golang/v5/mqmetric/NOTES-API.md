# Notes on the `mqmetric` package

This package was originally designed with the specific intention of supporting monitoring
programs that put MQ metrics into databases such as Prometheus. Those programs started off in
this repository. When they were moved to the `mq-metrics-samples` repository, it might have been
better to move this package with them. But that wasn't done. And it would be difficult to move them
now, as I know of some other monitoring solutions that have made use of the package.

So it was never intended to be a general purpose API, but builds on the core `ibmmq` API to
simplify a number of tasks needed for the monitors.

### Main APIs
The core public APIs are in `mqif` and `discover`.

* `mqif.go`: Manages the connection to a queue manager (via `InitConnection`) and can
return some information about the queue manager
  * InitConnection
  * InitConnectionKey
  * EndConnection
  * GetPlatform
  * GetCommandLevel
* `discover.go`: Handles the discovery of the metrics published by a queue manager, and then makes the
subscriptions to required topics. It also processes those publications, building maps containing the
various metrics and their values, tied to the object names.
  * VerifyConfig
  * DiscoverAndSubscribe
  * RediscoverAndSubscribe
  * RediscoverAttributes
  * ProcessPublications
  * Normalise
  * ReadPatterns
  * VerifyPattern
  * VerifyQueuePatterns
  * FilterRegExp
  * SetLocale
  * GetProcessPublicationCount
  * GetObjectDescription
  * GetDiscoveredQueues
* `globals.go`: Ways to access the metrics without directly referring to global variables. Added to deal
with the multiple connection redesign
   * GetObjectStatus
   * GetPublishedMetrics
   * SetConnectionKey
   * GetConnectionKey
* `<objecttype>.go`: Similar processing is available for each object type such as channel or queue. The
public APIs first initialise the status attributes explicitly supported (ie not coming from the published metadata)
and then collect the values. The xxNormalise functions ensure the metrics are in a suitable scale and datatype.
  * xxInitAttributes (eg UsageInitAttributes)
  * CollectXxStatus (eg CollectQueueStatus)
  * xxNormalise (eg ChannelNormalise)
  * InquireXxs (eg InquireTopics)
* `log.go`: The `SetLogger` function is called by a collector program to setup the output location for
error/info/trace logging.

### Multiple connection support
The code was written with the intention of just handling a single connection to a single queue manager
and collecting its various metrics serially. The close integration of the collectors and this package led to
various global public variables that could be accessed directly.

The March 2021 iteration has extended the design to try to support
monitoring multiple queue managers from a single collector program. The API extensions may not be
the most natural way of doing it, but I tried to do it while maintaining compatibility. Essentially,
all of the global variables are now semi-hidden behind getter functions; the globals are still available
for now, but access is deprecated. If there is a future major version change, those globals will disappear.

The new design allows a collector to associate a connection with a key. There's still expected to be
serialisation for now, so I doubt that you could process metrics from several qmgrs in parallel, in different
threads. The fundamental new API in the package is `InitConnectionKey`. Once you've done that, then
before calling other APIs in the package such as DiscoverAndSubscribe, you call `SetConnectionKey` to get the
appropriate connection in place. Calling that function another time would let you switch to a
different connection.

I have thoughts on how the APIs can be extended or modified (breaking) to permit parallel collection,
but this first phase gets the core data structures moved into better places. And it's not clear how valuable
the extensions would be - how many people would be interested.
