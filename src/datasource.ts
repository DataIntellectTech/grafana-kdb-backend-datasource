import { DataSourceInstanceSettings } from '@grafana/data';
import { DataSourceWithBackend, getBackendSrv, getTemplateSrv, toDataQueryResponse } from '@grafana/runtime';
import {MyDataSourceOptions, MyQuery, MyVariableQuery} from './types';


export class DataSource extends DataSourceWithBackend<MyQuery, MyDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<MyDataSourceOptions>) {
    super(instanceSettings);
  }
  applyTemplateVariables(query: MyQuery) {
    const templateSrv = getTemplateSrv();
    return {
      ...query,
      queryText: query.queryText ? templateSrv.replace(query.queryText) : '',
    };

  }

  async metricFindQuery(query: MyVariableQuery, options?: any): Promise<any> {
    const templateSrv = getTemplateSrv();
    let timeout = parseInt(query.timeOut, 10)
    const body: any = {
      queries: [
        { datasourceId:this.id,
          orgId: this.id,
          queryText: query.queryText ? templateSrv.replace(query.queryText) : '',
          timeOut: timeout,
        }
      ]
    }

    const backendQuery = getBackendSrv()
      .datasourceRequest({
          url: '/api/ds/query',
          method: 'POST',
          data: body,
        }).then((response: any) => {
          let parsedResponse = toDataQueryResponse(response)
          let responseValues: any[] = []
          for (let frame in parsedResponse.data) {
            responseValues = responseValues.concat(parsedResponse.data[frame].fields[0].values.toArray().map((x: any) => { return {text: x} }))
            }
          return responseValues
        }).catch(err => {
          console.log(err)
          err.isHandled = true; // Avoid extra popup warning
          return ({ text:"ERROR"})
        });
    return backendQuery
  }
}
