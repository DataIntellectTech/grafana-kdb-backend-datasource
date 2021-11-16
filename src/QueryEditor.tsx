

import React, { ChangeEvent, PureComponent } from 'react';
import { LegacyForms } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from './datasource';
import { MyDataSourceOptions, MyQuery } from './types';

const { FormField } = LegacyForms;

type Props = QueryEditorProps<DataSource, MyQuery, MyDataSourceOptions>;

export class QueryEditor extends PureComponent<Props> {
  onQueryTextChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query } = this.props;
    onChange({ ...query, queryText: event.target.value });
  };

  onFieldChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query } = this.props;
    onChange({ ...query, field: event.target.value });
  };



  render() {
    const query = this.props.query;
    const { queryText, field } = query;
    console.log(this.props);


    return (
      <div className="gf-form">

        <FormField
          labelWidth={7}
          value={queryText || ''}
          onChange={this.onQueryTextChange}
          label="KBD Query"
          tooltip="Please enter a KBD Query"
        />

        <FormField
            labelWidth={5}
            value={field || ''}
            onChange={this.onFieldChange}
            label="Field"
            tooltip="Please enter a Field"
        />

      </div>
    );
  }
}
