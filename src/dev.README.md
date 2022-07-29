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
5. Uncomment and set "allow_loading_unsigned_plugins=aquaqanalytics-kdbbackend-datasource"