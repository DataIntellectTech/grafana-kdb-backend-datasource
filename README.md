# Grafana  KDB Backend Datasource

[![Build](https://github.com/grafana/grafana-starter-datasource-backend/workflows/CI/badge.svg)](https://github.com/grafana/grafana-datasource-backend/actions?query=workflow%3A%22CI%22)

## What is KDB Backend Datasource?

KDB Backend Datasource is a plugin that adds the ability to query KDB from Grafana. It also enables the use of alerting on the data. It supports the use of varibles in the dashboard.

## Getting started for users

Below gives instructions for using the plugin

### Adding a data source

1. Navigate to settings -> datasources
2. Click add datasource and navigate to kdb-backend-datasource
3. Enter the URL and Port, along with the username and password (if required - defaults to "") of your KDB instance.
4. Enter a timeout value in ms, defaultis 1000ms
5. Click save & test
6. An alert at the bottom should display: "kdb connected succesfully"

### Creating a dashboard

1. Navigate to create -> dashboard
2. Create an empty panel
3. Under the KDB Query field enter a valid KDB query </br> For example: ([] time:reverse .z.p-0D00:20*til 10;val:til 10)
4. Optional Timeout can be entered - default is 10,000 ms
5. Click the refresh dashboard in the top right, above the Panel
6. The data should be displayed on the panel
7. Click save and return to your dashboard
8. The refresh rate can be set from your dashboard, click refresh, select the drop down menu and set your refresh time.

### Logging
ToDo

## Getting started for developers

Below gives instructions for building the plugin in both development mode and production mode.

### Frontend

1. Install dependencies

   ```bash
   yarn install
   ```

2. Build plugin in development mode or run in watch mode

   ```bash
   yarn dev
   ```

   or

   ```bash
   yarn watch
   ```

3. Build plugin in production mode

   ```bash
   yarn build
   ```

### Backend

1. Update [Grafana plugin SDK for Go](https://grafana.com/docs/grafana/latest/developers/plugins/backend/grafana-plugin-sdk-for-go/) dependency to the latest minor version:

   ```bash
   go get -u github.com/grafana/grafana-plugin-sdk-go
   go mod tidy
   ```

2. Build backend plugin binaries for Linux, Windows and Darwin:

   ```bash
   mage -v
   ```

3. List all available Mage targets for additional commands:

   ```bash
   mage -l
   ```

### Setting Grafana to development mode

1. Navigate to your Grafana conf folder (grafana/conf)
2. Create a duplicate of sample.ini renaming it to custom.ini (or update your existing custom.ini)
3. Open custom.ini in your favorite editor
4. Uncomment "app_mode" (Note: comments are ";") and set it "app_mode = development"
5. Save your changes and restart Grafana, changes should be implemented

### Adding plugin to Grafana
1. Navigate to your Grafana conf folder (grafana/conf)
2. Create a duplicate of sample.ini renaming it to custom.ini (or update your existing custom.ini)
3. Open custom.ini in your favorite editor
4. Uncomment the "plugin" parameter and set it to the directory containing your plugin, This does not need to be altered if you are installing the plugin in the default location
5. Uncomment and set "allow_loading_unsigned_plugins=aqua-q-kdb-backend-datasource"

### Security
By default we pass an empty "" string for both the username and password, these can be overridden in the datasource settings.
TLS is also support - enable with the TLS Client Auth switch. Enter the client TLS Key and client TLS Cert into the fields provided. To skip server vertification of the TLS certificate use the Skip TLS Verify switch. A CA Certificate can be used if the "With CA Cert" switch is enabled - for running unsigned certs

### kdb+
The queries are passed to kdb as a two item list in the form, the query is excuted as follows: {[x] value x[`Query;`Query]}
The data is in a nested structure as follows:

| Dataset                           | Data                                                         |
| --------------------------------- | ------------------------------------------------------------ |
| AQUAQ_KDB_BACKEND_GRAF_DATASOURCE | 1F (Version)                                                 |
| Time                              | Date and Timestamp                                           |
| OrgID                             | 1                                                            | 
| Datasource                        | ID, Name, UID, URL, Updated, User                            | 
| User                              | UserName, UserEmail, USerLogin, UserRole                     |
| Query                             | RefID, Query, QueryType, MaxDataPoints, Interval, TimeRange  |
| Timeout                           | 1000                                                         |

### Alerts
Before creating an alert, create a contact point under alerting -> contact points. Then create a notification policy under Alerting -> notification policy. 

To create an alert on a panel, navigate to the relevant dashboard and choose the edit option from here navigate to the Alert menu. Fill out the relevant Rule name, type and folder. Enter your kdb query and run the queries. You will be able to use expressions to query the data from this query.
Next of all set the alert conditions, making sure to select the expression and to set the evaluate duration. Finally set the custom label in Alert details.

### Variables
Plugin will handle static and query variables. It also allows the user to chain querys

#### Static Variables
These can be entered under the Custom type using Grafana format, (e.g. comma separated list). Static variables can then be used in queries in the form: ${variable_name}

#### Query Variables
These can be entered under the Query Type, the query will run upon click away or when update is pressed. The timeout field does not have to be populated. The list of returned variables will be displayed at the bottom. Queries can also take in other variables as well. The usage is the same as static variables ${variable_name}

### Timezones

kdb stores its times in UTC format, it is advised to set the Dashboard to UTC as well. This can be done in Dashboard settings - Timezone

### Limitations
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
