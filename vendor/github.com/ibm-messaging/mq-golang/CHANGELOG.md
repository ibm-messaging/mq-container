# Changelog

## May 2018

* Corrected package imports
* Formatted go code with `go fmt`
* Rearranged this file
* Removed logging from golang package `mqmetric`
* Moved samples to a separate repository
* Added build scripts for `ibmmq` and `mqmetric` packages and `ibmmq` samples
* Added unit tests for `ibmmq` and `mqmetric` packages

## March 2018 - v1.0.0

* Added V9.0.5 constant definitions
* Changed #cgo directives for Windows now the compiler supports standard path names
* Added mechanism to set MQ userid and password for Prometheus monitor
* Released v1.0.0 of this repository for use with golang dependency management tools

## October 2017

* Added V9.0.4 constant definitions - now generated from original MQ source code
* Added MQSC script to show how to redefine event queues for pub/sub
* Prometheus collector has new parameter to override the first component of the metric name
* Prometheus collector can now process channel-level statistics

## 18 May 2017

* Added the V9.0.3 constant definitions.
* Reinstated 64-bit structure "length" fields in cmqc.go after fixing a bug in the base product C source code generator.

## 25 Mar 2017

* Added the metaPrefix option to the Prometheus monitor. This allows selection of non-default resources such as the MQ Bridge for Salesforce included in MQ 9.0.2.

## 15 Feb 2017

* API BREAKING CHANGE: The MQI verbs have been changed to return a single error indicator instead of two separate values. See mqitest.go for examples of how MQRC/MQCC codes can now be tested and extracted. This change makes the MQI implementation a bit more natural for Go environments.

## 10 Jan 2017

* Added support for the MQCD and MQSCO structures to allow programmable client connectivity, without requiring a CCDT. See the clientconn sample program for an example of using the MQCD.
* Moved sample programs into subdirectory

## 14 Dec 2016

* Minor updates to this README for formatting
* Removed xxx_CURRENT_LENGTH definitions from cmqc

## 07 Nov 2016

* Added a collector that prints metrics in a simple JSON format. See the [README](cmd/mq_json/README.md) for more details.
* Fixed bug where freespace metrics were showing as non-integer bytes, not percentages

## 17 Oct 2016

* Added some Windows support. An example batch file is included in the mq_influx directory; changes would be needed to the MQSC script to call it. The other monitor programs can be supported with similar modifications.
* Added a "getting started" section to this README.

## 23 Aug 2016

* Added a collector for Amazon AWS CloudWatch monitoring. See the [README](cmd/mq_aws/README.md) for more details.

## 12 Aug 2016

* Added a OpenTSDB monitor. See the [README](cmd/mq_opentsdb/README.md) for more details.
* Added a Collectd monitor. See the [README](cmd/mq_coll/README.md) for more details.
* Added MQI MQCNO/MQCSP structures to support client connections and password authentication with MQCONNX.
* Allow client-mode connections from the monitor programs
* Added Grafana dashboards for the different monitors to show how to query them
* Changed database password mechanism so that "exec" maintains the PID for MQ services

## 04 Aug 2016

* Added a monitor command for exporting MQ data to InfluxDB. See the [README](cmd/mq_influx/README.md) for more details
* Restructured the monitoring code to put common material in the mqmetric package, called from the Influx and Prometheus monitors.

## 25 Jul 2016

* Added functions to handle basic PCF creation and parsing
* Added a monitor command for exporting MQ V9 queue manager data to Prometheus. See the [README](cmd/mq_prometheus/README.md) for more details

## 18 Jul 2016

* Changed structures so that most applications will not need to use cgo to imbed the MQ C headers
  * Go programs will now use int32 where C programs use MQLONG
  * Use of message handles, distribution lists require cgo for now
* Package ibmmq now includes the numeric #defines as a Go file, cmqc.go, for easier use
* Removed "src/" prefix from tree in github repo
* Removed need for buffer length parm on Put/Put1
* Updated comments
* Added MQINQ
* Added MQItoString function for some maps of values to constant names

## 08 Jul 2016

* Initial release