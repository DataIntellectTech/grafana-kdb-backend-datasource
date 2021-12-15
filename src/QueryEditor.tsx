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

    onTimeOutChange = (event: ChangeEvent<HTMLInputElement>) => {
        if((/^\d+$/.test(event.target.value) || event.target.value==="")){
            const { onChange, query } = this.props;
            onChange({ ...query, timeOut: parseInt(event.target.value, 10)});
        }
    };



    render() {
        const query = this.props.query;
        const { queryText, timeOut } = query;
        return (
            <>
                <div style={{paddingBottom: 10}}>
                <FormField
                    inputWidth={40}
                    labelWidth={8}
                    value={queryText || ''}
                    onChange={this.onQueryTextChange}
                    label="KDB Query"
                    tooltip="Please enter a KDB Query"
                />
                </div>

                <div style={{paddingBottom: 10}}>
                <FormField
                    inputWidth={15}
                    labelWidth={8}
                    value={timeOut || ''}
                    onChange={this.onTimeOutChange}
                    label="Timeout (ms)"
                    tooltip="Please enter a Timeout in ms, default is 10,000 ms"
                />
                </div>


                </>
        );
    }
}
