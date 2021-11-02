import React, { ChangeEvent, PureComponent } from 'react';
import { LegacyForms} from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { MyDataSourceOptions, MySecureJsonData } from './types';

const { FormField } = LegacyForms;

interface Props extends DataSourcePluginOptionsEditorProps<MyDataSourceOptions> {}

interface State {}

export class ConfigEditor extends PureComponent<Props, State> {



  onHostChange = (event: ChangeEvent<HTMLInputElement>) => {

    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      host: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  }

  onPortChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      port: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  }

  onUsernameChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const { secureJsonData } = options;
    onOptionsChange({
      ...options,
      secureJsonData: {
        ...secureJsonData,
        username: event.target.value,
      },
    });
  };

  onPasswordChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const { secureJsonData } = options;
    onOptionsChange({
      ...options,
      secureJsonData: {
        ...secureJsonData,
        password: event.target.value,
      },
    });
  };

  render() {
    const { options } = this.props;
    const { jsonData } = options;
    const secureJsonData = (options.secureJsonData || {}) as MySecureJsonData;
    console.log(this.props);
    return (
        <div className="gf-form-group">

          <div className="gf-form">
            <FormField
                label="Host"
                labelWidth={6}
                inputWidth={20}
                onChange={this.onHostChange}
                value={jsonData.host || ''}
                placeholder="json field returned to frontend"
            />
          </div>
          <div className="gf-form">
            <FormField
                label="Port"
                labelWidth={6}
                inputWidth={20}
                onChange={this.onPortChange}
                value={jsonData.port || ''}
                placeholder="json field returned to frontend"
            />
          </div>

          <div className="gf-form">
            <FormField
                label="Username"

                labelWidth={6}
                inputWidth={20}
                onChange={this.onUsernameChange}
                value={secureJsonData.username || ''}
                placeholder="json field returned to frontend"
            />
          </div>

          <div className="gf-form">
            <FormField
                label="Password"

                labelWidth={6}
                inputWidth={20}
                onChange={this.onPasswordChange}
                value={secureJsonData.password || ''}
                placeholder="json field returned to frontend"
            />
          </div>


        </div>
    );
  }
}
