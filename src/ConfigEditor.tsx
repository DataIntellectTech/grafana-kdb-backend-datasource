import React, { ChangeEvent, PureComponent } from 'react';
import { LegacyForms} from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { MyDataSourceOptions, MySecureJsonData } from './types';

const { FormField, SecretFormField, Switch } = LegacyForms;

interface Props extends DataSourcePluginOptionsEditorProps<MyDataSourceOptions> {}

interface State {}

export class ConfigEditor extends PureComponent<Props, State> {

  state = {
    displayTLS:false
  }

  onHostChange = (event: ChangeEvent<HTMLInputElement>) => {

    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      host: event.target.value,
    };
    // @ts-ignore
    onOptionsChange({ ...options, jsonData });
  }

  onPortChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;

    if((/^\d+$/.test(event.target.value) || event.target.value==="")){
      const jsonData = {
        ...options.jsonData,
        port: parseInt(event.target.value),
      };
      onOptionsChange({ ...options, jsonData });
    }
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

  onResetUsername = () => {
    const { onOptionsChange, options } = this.props;
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        username: false,
      },
      secureJsonData: {
        ...options.secureJsonData,
        username: '',
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

  onResetPassword = () => {
    const { onOptionsChange, options } = this.props;
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        password: false,
      },
      secureJsonData: {
        ...options.secureJsonData,
        password: '',
      },
    });
  };

  onTlsToggle = () => {
    this.setState({displayTLS:!this.state.displayTLS});
    const { onOptionsChange, options } = this.props;
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        tlsKey: false,
        tlsCertificate: false,
      },
      secureJsonData: {
        ...options.secureJsonData,
        tlsKey: '',
        tlsCertificate: '',
      },
    });
  };

  onTlsCertificateChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const { secureJsonData } = options;
    onOptionsChange({
      ...options,
      secureJsonData: {
        ...secureJsonData,
        tlsCertificate: event.target.value,
      },
    });
  };

  onTlsCertificateReset = () => {
    const { onOptionsChange, options } = this.props;
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        tlsCertificate: false,
      },
      secureJsonData: {
        ...options.secureJsonData,
        tlsCertificate: '',
      },
    });
  };

  onTlsKeyChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const { secureJsonData } = options;
    onOptionsChange({
      ...options,
      secureJsonData: {
        ...secureJsonData,
        tlsKey: event.target.value,
      },
    });
  };

  onTlsKeyReset = () => {
    const { onOptionsChange, options } = this.props;

    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        tlsKey: false,
      },
      secureJsonData: {
        ...options.secureJsonData,
        tlsKey: '',
      },
    });
  };

  renderTLS = () => {
    const { options } = this.props;
    const { secureJsonFields } = options;
    const secureJsonData = (options.secureJsonData || {}) as MySecureJsonData;

    return (
        <>
          <div className="gf-form">
            <SecretFormField
                isConfigured={(secureJsonFields && secureJsonFields.tlsKey) as boolean}
                value={secureJsonData.tlsKey || ''}
                label="TLS Key"
                placeholder="TLS Key"
                labelWidth={7}
                inputWidth={20}
                onReset={this.onTlsKeyReset}
                onChange={this.onTlsKeyChange}
            />
          </div>
          <div className="gf-form">
            <SecretFormField
                isConfigured={(secureJsonFields && secureJsonFields.tlsCertificate) as boolean}
                value={secureJsonData.tlsCertificate || ''}
                label="TLS Certificate"
                placeholder="TLS Certificate"
                labelWidth={7}
                inputWidth={20}
                onReset={this.onTlsCertificateReset}
                onChange={this.onTlsCertificateChange}
            />
          </div>
        </>
    )
  }

  render() {
    const { options } = this.props;
    const { jsonData, secureJsonFields } = options;
    const secureJsonData = (options.secureJsonData || {}) as MySecureJsonData;
    return (
        <div className="gf-form-group">

          <div className="gf-form">
            <FormField
                label="Host"
                labelWidth={7}
                inputWidth={20}
                onChange={this.onHostChange}
                value={jsonData.host || ''}
                placeholder="Please enter host URL"
            />
          </div>
          <div className="gf-form">
            <FormField
                label="Port"
                labelWidth={7}
                inputWidth={20}
                onChange={this.onPortChange}
                value={jsonData.port || ''}
                placeholder="Please enter host port"
            />
          </div>

          <div className="gf-form">

            <SecretFormField
                isConfigured={(secureJsonFields && secureJsonFields.username) as boolean}
                value={secureJsonData.username || ''}
                label="Username"
                placeholder="Username"
                labelWidth={7}
                inputWidth={20}
                onReset={this.onResetUsername}
                onChange={this.onUsernameChange}
            />

          </div>

          <div className="gf-form">
            <SecretFormField
                isConfigured={(secureJsonFields && secureJsonFields.password) as boolean}
                value={secureJsonData.password || ''}
                label="Password"
                placeholder="Password"
                labelWidth={7}
                inputWidth={20}
                onReset={this.onResetPassword}
                onChange={this.onPasswordChange}
            />
          </div>
          {this.state.displayTLS && <>{this.renderTLS()}</>}
          <div className="gf-form">
          <Switch checked={this.state.displayTLS} label="Enable TLS" onChange={this.onTlsToggle} />
          </div>


        </div>
    );
  }
}
