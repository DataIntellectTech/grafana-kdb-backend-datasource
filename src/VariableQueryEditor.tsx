import React, {ChangeEvent, useState} from 'react';
import { MyVariableQuery } from './types';

interface VariableQueryProps {
    query: MyVariableQuery;
    onChange: (query: MyVariableQuery, definition: string) => void;
}

export const VariableQueryEditor: React.FC<VariableQueryProps> = ({ onChange, query }) => {
    const [state, setState] = useState(query);

    const saveQuery = () => {
        onChange(state, `${state.queryText} (${state.timeOut})`);
    };

    const handleTimeOutChange = (event: ChangeEvent<HTMLInputElement>) =>{
    if((/^\d+$/.test(event.target.value) || event.target.value==="")){
        console.log(event.target.value)
        setState({
            ...state,
            timeOut:event.target.value,

        })}};

    const handleQueryChange = (event: React.FormEvent<HTMLInputElement>) =>
        setState({
            ...state,
            queryText: event.currentTarget.value,

        });

    return (
        <>
            <div className="gf-form">
                <span className="gf-form-label width-10">Query</span>
                <input
                    name="queryText"
                    className="gf-form-input"
                    onBlur={saveQuery}
                    onChange={handleQueryChange}
                    value={state.queryText}
                />
            </div>
            <div className="gf-form">
                <span className="gf-form-label width-10">Timeout</span>
                <input
                    name="timeOut"
                    className="gf-form-input"
                    onBlur={saveQuery}
                    onChange={handleTimeOutChange}
                    value={state.timeOut}
                />
            </div>

        </>
    );
};