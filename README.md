# Grafana  KDB Backend Datasource

[![Build](https://github.com/grafana/grafana-starter-datasource-backend/workflows/CI/badge.svg)](https://github.com/grafana/grafana-datasource-backend/actions?query=workflow%3A%22CI%22)

## What is KDB Backend Datasource?

KDB Backend Datasource is a plugin that adds the ability to query KDB from Grafana. It also enables the use of alerting on the data. It supports the use of varibles in the dashboard.

## Getting started for users

Below gives instructions for using the plugin

### Adding a data source

1. Navigate to settings -> datasources
2. Click add datasource and navigate to kdb-backend-datasource
3. Enter the URL and Port of your KDB instance. A username and password must be entered for the KDB instance. <br /> Note: If your instance doesn't have a username or password, fill these fields with random text <br /> For example: username: Ausername, password: Apassword
4. Enter a timeout value in ms, suggested is 350
5. Click save & test
6. An alert at the bottom should display: "kdb connected succesfully"

### Creating a dashboard

1. Navigate to create -> dashboard
2. Create an empty panel
3. Under the KDB Query field enter a valid KDB query </br> For example: ([] time:reverse .z.p-0D00:20*til 10;val:til 10)
4. Click the refresh dashboard in the top right, above the Panel
5. The data should be displayed on the panel
6. Click save and return to your dashboard
7. The refresh rate can be set from your dashboard, click refresh, select the drop down menu and set your refresh time.


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

