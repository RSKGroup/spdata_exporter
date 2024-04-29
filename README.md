# spdata_exporter
MacOS system_profiler exporter for Prometheus. Please note, due to the ever-changing nature and depth of these metrics, they are presented as labels.
## Table of Contents
- [Compatibility](#compatibility)
- [Dependency](#dependency)
- [Download](#download)
- [Compile](#compile)
- [Run](#run)
- [Flags](#flags)
- [Metrics](#metrics)
- [Contribute](#contribute)
- [License](#license)

Compatibility
-------------
Supports macOS 13+ for both Intel and Apple Silicon Chipset Archtectures
Dependency

Download
--------

Compile
-------
1. Install the XCode Command Line Tools
2. Install go 1.20+
3. Clone the repository
4. Build your version:
  - `GOOS=darwin GOARCH=amd64 go build -o spdata_exporter-amd64`
  - `GOOS=darwin GOARCH=arm64 go build -o spdata_exporter-arm64`
6. If desired, create a universal with
  - `lipo -create -output spdata_exporter spdata_exporter-amd64 spdata_exporter-arm64`
7. Sign whichever version or the universal with
  - `codesign --deep --force --verbose --sign "your-developer-cert" your-executable`

Run
---
```shell
spdata_exporter --config /usr/local/etc/spdata_exporter.yml
```

Flags
-----
TThe only launch flag is to identify the configuration file as presented in [Run](#run)

Metrics
-------
spdata_exporter provides metrics from all system_profiler data types visible through system_profiler -ListDataTypes including:
<pre>
Available Datatypes:
SPParallelATADataType
SPUniversalAccessDataType
SPSecureElementDataType
SPApplicationsDataType
SPAudioDataType
SPBluetoothDataType
SPCameraDataType
SPCardReaderDataType
SPiBridgeDataType
SPDeveloperToolsDataType
SPDiagnosticsDataType
SPDisabledSoftwareDataType
SPDiscBurningDataType
SPEthernetDataType
SPExtensionsDataType
SPFibreChannelDataType
SPFireWireDataType
SPFirewallDataType
SPFontsDataType
SPFrameworksDataType
SPDisplaysDataType
SPHardwareDataType
SPInstallHistoryDataType
SPInternationalDataType
SPLegacySoftwareDataType
SPNetworkLocationDataType
SPLogsDataType
SPManagedClientDataType
SPMemoryDataType
SPNVMeDataType
SPNetworkDataType
SPPCIDataType
SPParallelSCSIDataType
SPPowerDataType
SPPrefPaneDataType
SPPrintersSoftwareDataType
SPPrintersDataType
SPConfigurationProfileDataType
SPRawCameraDataType
SPSASDataType
SPSerialATADataType
SPSPIDataType
SPSmartCardsDataType
SPSoftwareDataType
SPStartupItemDataType
SPStorageDataType
SPSyncServicesDataType
SPThunderboltDataType
SPUSBDataType
SPNetworkVolumeDataType
SPWWANDataType
SPAirPortDataType
</pre>

Each data type is parsed as a gauged label thusly:
<pre>
# HELP spdata_spstoragedatatype Metric spdata_spstoragedatatype dynamically created
# TYPE spdata_spstoragedatatype gauge
spdata_spstoragedatatype{device="0",name="_name",value="Macintosh HD - Data"} 1
spdata_spstoragedatatype{device="0",name="bsd_name",value="disk1s1"} 1
spdata_spstoragedatatype{device="0",name="file_system",value="APFS"} 1
spdata_spstoragedatatype{device="0",name="free_space_in_bytes",value="1.877371875328e+12"} 1
spdata_spstoragedatatype{device="0",name="ignore_ownership",value="no"} 1
spdata_spstoragedatatype{device="0",name="mount_point",value="/System/Volumes/Data"} 1
</pre>

Additionally, this tool exports the following data:
- Core File Count and Cores from FSM
- cvlabel -l|wc -l count
- Latest Time Machine backup time
- NTP date including NTP Server, Status, and Time Zone
<pre>
# HELP spdata_corefilescount Metric spdata_corefilescount dynamically created
# TYPE spdata_corefilescount gauge
spdata_corefilescount{device="0",name="fsm",value="0"} 0
spdata_corefilescount{device="0",name="total",value="0"} 0
# HELP spdata_cvlabelcount Metric spdata_cvlabelcount dynamically created
# TYPE spdata_cvlabelcount gauge
spdata_cvlabelcount{device="0",name="cvlabel",value="22"} 22
# HELP spdata_latestbackuptime Metric spdata_latestbackuptime dynamically created
# TYPE spdata_latestbackuptime gauge
spdata_latestbackuptime{device="0",name="latestbackup",value="2024-02-19-111614"} 1
# HELP spdata_ntpserver Metric spdata_ntpserver dynamically created
# TYPE spdata_ntpserver gauge
spdata_ntpserver{device="0",name="ntpserver",value="time.apple.com"} 1
spdata_ntpserver{device="0",name="ntpstatus",value="On"} 1
spdata_ntpserver{device="0",name="timezone",value="America/Los_Angeles"} 1
</pre>

Contribute
----------
If you like system_profiler Exporter, please give us a star. This will help more people know system_profiler Exporter.

Please feel free to send me [pull requests](https://github.com/rskgroup/spdata_exporter/pulls).

License
-------
Code is licensed under the [Apache License 2.0](https://github.com/danielqsj/kafka_exporter/blob/master/LICENSE).
