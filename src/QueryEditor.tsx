import React, { ChangeEvent, PureComponent, SyntheticEvent } from 'react';
import { InlineFieldRow, InlineField, LegacyForms, Input, InlineSwitch } from '@grafana/ui';
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
            onChange({ ...query, timeOut: parseInt(event.target.value, 10) });
        }
    };
    onUseTimeColumnToggle = (event: SyntheticEvent<HTMLInputElement, Event>) => {
        const { onChange, query } = this.props;
        onChange({ ...query, useTimeColumn: !query.useTimeColumn });
    };
    onTimeColumnChange = (event: ChangeEvent<HTMLInputElement>) => {
        if((/^\d+$/.test(event.target.value) || event.target.value==="")){
            const { onChange, query } = this.props;
            onChange({ ...query, timeColumn: event.target.value });
        }
    };
    onIncludeKeyColumnsToggle = (event: SyntheticEvent<HTMLInputElement, Event>) => {
        const { onChange, query } = this.props;
        onChange({ ...query, includeKeyColumns: !query.includeKeyColumns });
    };

    render() {
        const query = this.props.query;
        const { queryText, timeOut, useTimeColumn, includeKeyColumns, timeColumn } = query;
        return (
            <>
                <div style={{paddingBottom: 4}}>
                <FormField
                    name="QueryTextInputField"
                    inputWidth={40}
                    labelWidth={13}
                    value={queryText || ''}
                    onChange={this.onQueryTextChange}
                    label="KDB Query"
                    tooltip="Please enter a KDB Query"
                />
                </div>
                <div style={{paddingBottom: 4}}>
                <FormField
                    name="TimeoutTextInputField"
                    inputWidth={15}
                    labelWidth={13}
                    value={timeOut || ''}
                    onChange={this.onTimeOutChange}
                    label="Timeout (ms)"
                    tooltip="Please enter a Timeout in ms, default is 10,000 ms"
                />
                </div>
                <div style={{paddingBottom: 4}}>
                <InlineFieldRow>
                    <InlineField
                        label="Use Custom Time Column"
                        labelWidth={26}
                        tooltip="Select to use a custom temporal column as the time axis"
                        >
                        <InlineSwitch checked={useTimeColumn} label="Use Custom Time Column" onChange={this.onUseTimeColumnToggle}/>
                    </InlineField>
                    <InlineField
                        hidden={!useTimeColumn}
                        label="Time Column"
                        labelWidth={26}
                        tooltip="Name of temporal column to use as the time axis"
                        >
                        <Input
                            hidden={!useTimeColumn}
                            className="TimeColumnInputField"
                            width={30}
                            value={timeColumn || ''}
                            onChange={this.onTimeColumnChange}
                        />
                    </InlineField>
                </InlineFieldRow>
                <InlineField
                    label="Include Keys In Output"
                    labelWidth={26}
                    tooltip="If enabled, key columns will be projected and included in the output for grouped-series results">
                    <InlineSwitch checked={includeKeyColumns} onChange={this.onIncludeKeyColumnsToggle} />
                </InlineField>
                </div>
                </>
        );
    }
}
