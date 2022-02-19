# Grafana  KDB+ Backend Datasource

[![Build](https://github.com/grafana/grafana-starter-datasource-backend/workflows/CI/badge.svg)](https://github.com/grafana/grafana-datasource-backend/actions?query=workflow%3A%22CI%22)

## What is KDB+ Backend Datasource?

KDB+ Backend Datasource is a plugin that adds the ability to query KDB+ from Grafana. It also enables the use of alerting on the data. It supports the use of varibles in the dashboard.

## Getting started for users

Below gives instructions for using the plugin

### Adding a data source

1. Navigate to settings -> datasources
2. Click add datasource and navigate to kdb-backend-datasource
3. Enter the URL and Port, along with the username and password of your KDB+ instance (if required - if not supplied these will default to `""`).
4. Enter a timeout value (in ms), default is `1000` miliseconds
5. Click save & test
6. An alert at the bottom should display: "kdb+ connected succesfully"

### Creating a dashboard

1. Navigate to `Create - Dashboard` from the toolbar on the left.
2. Create an empty panel. New dashboards will have an empty panel already present. New panels can be added with the `Add panel` button in the top-right taskbar.
3. Under the KDB+ Query field enter a valid KDB+ query e.g. `([] time:reverse .z.p-0D00:05*til 20;val:til 20)` .
4. Optionally a custom timeout can be defined in the `Timeout (ms)` entry field. The default is `10 000` ms.
5. Click the `Refresh dashboard` button in the top right, above the Panel visualisation. The data should appear in the visualisation.
6. Click the `Go back (Esc)` button in the top-left of the page to exit the query editor and return to the dashboard.
7. The dashboard can be saved with the `Save dashboard` button in the top right. If required, an automated refresh-rate can be set for the dashboard, click the drop-down menu next to the `Refresh dashboard` button in the top-right and set your desired refresh-rate. Custom refresh-rates can be added in `Dashboard settings` .

## Variables
This plugin can handle static and query variables. It also allows the user to chain queries.

### Static & Multi-Value Variables
These can be entered under the `Custom` variable type using Grafana's standard format, (i.e. comma separated list). Static variables can then be used in queries in the form: 

`${variable_name}`

If using `Multi-value` variables where more than one value can be selected at a time, please refer to [Grafana's documentation on formatting variables](https://grafana.com/docs/grafana/latest/variables/advanced-variable-format-options). We would recommend injecting `Multi-value` variables in the `csv` format, then splitting and casting these to a list of the required type in kdb+ with the [`sv`](https://code.kx.com/q/ref/sv/) operator, such as in the following `select` statement:

``select from trade where exchange in `$"," vs "${multi_variable_name:csv}"``

### Temporal Variables
Temporal variables (e.g. `${__from}` , `${__to}` etc) are injected by Grafana as the number of ***miliseconds*** since the Unix epoch (e.g. `1594671549254` corresponds to `Jul 13 2020 20:19:09`). To use temporal variables in kdb+ we need to manipulate these to match kdb+'s accepted formats.

This can be done by adjusting from *miliseconds* to *seconds* by dividing by 1000, converting to a decimal string with [`.Q.f`](https://code.kx.com/q/ref/dotq/#qf-format) and then [`tokking`](https://code.kx.com/q/ref/tok/) to a timestamp: 

``("P"$.Q.f[3] ${__from}%1000)``

Alternatively Grafana-injected temporal variables can be formatted to the [`ISO-8601`](https://www.iso.org/iso-8601-date-and-time-format.html) format and this can be `tokked` to a `datetime` datatype in kdb+:

``("Z"$"${__from:date}")``

As of kdb+ version 4.0 `ISO-8601` formatted date-times cannot be directly `tokked` to timestamps.

If the `timestamp` being referenced only requires accuracy to a single second, then they can be injected directly in unix-time and `tokked` to timestamps:

``("P"$"${__from:date:seconds}")``

### Query Variables
These can be entered under the `Query` variable type. These variables run a query against the target datasource before the panel queries are run, and from this meta-query builds a variable/list of variables. The query will run when the `Update` button is pressed. There is an optional `Timeout` field which if not defined will default to `10 000` ms. The list of returned variables will be displayed at the bottom of this page after the list is updated. 

Query variables can also take in other variables as part of the query which they run (sometimes called `Chained` variables). The format for this is the same as static & multi-value variables (`${variable_name}`)

## Security
By default we pass an empty `""` string for both the username and password, these can be overridden in the datasource settings.
TLS is also supported - enable with the `TLS Client Auth` switch. Enter the client TLS key and client TLS cert into the fields provided. To skip server vertification of the TLS certificate use the `Skip TLS Verify` switch. A custom Certificate Authority certificate can be used if the `With CA Cert` switch is enabled - use if the kdb+ datasource is running a custom-signed certificate.

## kdb+
The queries are passed to kdb+ as a two item synchronous query in the following kdb+ form:

``({[x] value x[`Query;`Query]};**QUERYDATA**)``

The `**QUERYDATA**` is a dictionary (kdb+ type `99`) with a nested structure as follows:

| Key                               | Value (`kdb+ type`)                                          |
| --------------------------------- | ------------------------------------------------------------ |
| AQUAQ_KDB_BACKEND_GRAF_DATASOURCE | Plugin Version (`float atom`)                                |
| Time                              | Query Timestamp (`timestamp atom`)                           |
| OrgID                             | Grafana Organisation ID (`long atom`)                        | 
| Datasource                        | **Datasource Info Object** (`dictionary`)                    | 
| User                              | **User Info Object** (`dictionary`)                          |
| Query                             | **Query Info Object** (`dictionary`)                         |
| Timeout                           | Grafana-side timeout duration in ms (`long atom`)            |

### **Datasource Info Object**

| Key | Value (`kdb+ type`) |
|-----|---------------------|
| ID | Datasource ID assigned by the Grafana instance (`long atom`) |
| Name | Name of the datasource as assigned by the user (`char list`) |
| UID | Datasource UID assigned by the Grafana instance (`char list`) |
| Updated | Timestamp of when the datasource was last updated (`timestamp atom`) |
| URL | URL of the datasource (?) (`char list`) | 
| User | `UserName` of User who created the datasource (`char list`)


### **User Info Object**

| Key | Value (`kdb+ type`) |
| ----|-------|
| UserName | User's Grafana *name* (not login username) (`char list`)|
| UserEmail | User's Grafana email address (`char list`)|
| UserLogin | User's Grafana login username (`char list`)|
| UserRole  | User's Grafana role (`char list`)|

### **Query Info Object**

**N.B. The `RefID`, `MaxDataPoints`, `Interval` and `TimeRange` keys are not present in `HEALTHCHECK` type queries.**

| Key | Value (`kdb+ type`) |
|-----|---------------------|
| RefID | Ref ID of query (`char list`) |
| Query | Query string which is evaluated (`char list`) |
| QueryType | Query *type* (`HEALTHCHECK` or `QUERY`) (`symbol atom`) |
| MaxDataPoints | Panel's defined max data-points (currently unused) (`long atom`)|
| Interval | Panel's defined interval (currently unused) (`long atom`) |
| TimeRange | `__from` and `__to` time range of query (`2 item timestamp list`) |

## Alerts
Before creating an alert, create a contact point under alerting -> contact points. Then create a notification policy under Alerting -> notification policy.

To create an alert on a panel, navigate to the relevant dashboard and choose the edit option from here navigate to the Alert menu. Fill out the relevant Rule name, type and folder. Enter your kdb+ query and run the queries. You will be able to use expressions to query the data from this query.
Next of all set the alert conditions, making sure to select the expression and to set the evaluate duration. Finally set the custom label in Alert details.

## Timezones

kdb+ stores its times in UTC format, it is advised to set the Dashboard to UTC as well. This can be done in Dashboard settings - Timezone

## Limitations
Infinities and nulls in Grafana do not share same data type as in kdb+.  An underlying string value representation is displayed rather than the null or infinity value held in kdb+. It is recommended Grafana send users handle null representations as per their data schema, data dictionary.

### Columns
The columns must be of constant type - there cannot be mixed lists as columns, (excluding 'string' columns).

#### Grouped Tables
Grouped tables are handled as follows: each grouping is returned as a frame, the name of the frame is the string representation of the column names seperated by a semicolon.

#### Nulls and Infinities
The table below displays how nulls, infinities and zeroes are handled for each data type:

| Field  | Short  | Int         | Long                 | Chars       | Symbols     | Timestamps  | Times       | Datetimes   | Timespans   | Months      | Dates       | Minutes     | Seconds     |
| ------ | ------ | ----------- | -------------------- | ----------- | ----------- | ----------- | ----------- | ----------- | ----------- | ----------- | ----------- | ----------- | ----------- |
| Zero   | 0      | 0           | 0                    | 0           | 0           | 0           | 0           | 0           | 0           | 0           | 0           | 0           | 0           |
| Null   | -32768 | -2147483648 | -9223372036854776000 | -2147483648 | -2147483648 | -2147483648 | -2147483648 | -2147483648 | -2147483648 | -2147483648 | -2147483648 | -2147483648 | -2147483648 |
| NegInf | -32767 | -2147483647 | -9223372036854776000 | -2147483647 | -2147483647 | -2147483647 | -2147483647 | -2147483647 | -2147483647 | -2147483647 | -2147483647 | -2147483647 | -2147483647 |
| Inf    | 32767  | 2147483647  | 9223372036854776000  | 2147483647  | 2147483647  | 2147483647  | 2147483647  | 2147483647  | 2147483647  | 2147483647  | 2147483647  | 2147483647  | 2147483647  |