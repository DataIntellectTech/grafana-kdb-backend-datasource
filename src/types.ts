import { DataQuery, DataSourceJsonData } from '@grafana/data';

export interface MyQuery extends DataQuery {
  queryText?: string;
  field: string;
}

/**
 * These are options configured for each DataSource instance.
 */
export interface MyDataSourceOptions extends DataSourceJsonData {
  host: string;
  port: number;
  timeout?: string;
  withTLS: boolean;
  skipVerifyTLS: boolean;
  withCACert: boolean;

}

export const defaultConfig: Partial<MyDataSourceOptions> = {
  withTLS: false,
  skipVerifyTLS: false,
  withCACert: false,

};

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface MySecureJsonData {
  username: string;
  password: string;
  tlsCertificate?:string;
  tlsKey?:string;
  caCert?:string;
}
