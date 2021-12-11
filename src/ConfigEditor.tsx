import { defaults } from 'lodash';
import React, { ChangeEvent, PureComponent } from 'react';
import { LegacyForms} from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import {defaultConfig, MyDataSourceOptions, MySecureJsonData} from './types';
// @ts-ignore
import { version } from '../package.json';
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
        port: parseInt(event.target.value, 10),
      };
      onOptionsChange({ ...options, jsonData });
    }
  }
  onTimeoutChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;

    if((/^\d+$/.test(event.target.value) || event.target.value==="")){
      const jsonData = {
        ...options.jsonData,
        timeout: event.target.value,
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

    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      withTLS: !options.jsonData.withTLS
    };
    // @ts-ignore
    onOptionsChange({ ...options, jsonData });
  };

  onSkipTlsToggle = () => {

    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      skipVerifyTLS: !options.jsonData.skipVerifyTLS
    };
    // @ts-ignore
    onOptionsChange({ ...options, jsonData });
  };

  onCaCertToggle = () => {

    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      withCACert: !options.jsonData.withCACert
    };
    // @ts-ignore
    onOptionsChange({ ...options, jsonData });
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
  onCaCertChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const { secureJsonData } = options;
    onOptionsChange({
      ...options,
      secureJsonData: {
        ...secureJsonData,
        caCert: event.target.value,
      },
    });
  };

  onCaCertReset = () => {
    const { onOptionsChange, options } = this.props;

    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        caCert: false,
      },
      secureJsonData: {
        ...options.secureJsonData,
        caCert: '',
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
                name="TLSKeyInputField"
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
                name="TLSCertInputField"
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
          {options.jsonData.withCACert &&
          <div className="gf-form">
            <SecretFormField
                name="TLSCAInputField"
                isConfigured={(secureJsonFields && secureJsonFields.caCert) as boolean}
                value={secureJsonData.caCert || ''}
                label="CA Certificate"
                placeholder="CA Certificate"
                labelWidth={7}
                inputWidth={20}
                onReset={this.onCaCertReset}
                onChange={this.onCaCertChange}
            />
          </div>}


        </>
    )
  }

  render() {
    const { options } = defaults(this.props, defaultConfig);
    const { jsonData, secureJsonFields } = options;
    const secureJsonData = (options.secureJsonData || {}) as MySecureJsonData;
    return (
        <div className="gf-form-group">

          <div className="gf-form">
            <FormField
                name="HostInputField"
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
                name="PortInputField"
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
                name="UsernameInputField"
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
                name="PasswordInputField"
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
          {!options.jsonData.withTLS &&
          <div className="gf-form">
            <FormField
                name="TimeoutInputField"
                label="Timeout"
                labelWidth={7}
                inputWidth={20}
                onChange={this.onTimeoutChange}
                value={jsonData.timeout || ''}
                placeholder="Please set timeout"
            />
          </div>}


          {options.jsonData.withTLS && <>{this.renderTLS()}</>}
          <div className="gf-form">
          <Switch checked={options.jsonData.withTLS} label="TLS Client Auth" onChange={this.onTlsToggle} />
            {options.jsonData.withTLS && <>
            <Switch checked={options.jsonData.skipVerifyTLS} label="Skip TLS Verify" onChange={this.onSkipTlsToggle} />
            <Switch checked={options.jsonData.withCACert} label="With CA Cert" onChange={this.onCaCertToggle} />
            </>}
          </div>

          <div className="gf-form">
            Version: {version}
          </div>


        </div>
    );
  }
}
