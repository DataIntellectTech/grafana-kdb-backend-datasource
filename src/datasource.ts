import { DataSourceInstanceSettings } from '@grafana/data';
import { DataSourceWithBackend, getBackendSrv, getTemplateSrv } from '@grafana/runtime';
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

    let timeout = parseInt(query.timeOut, 10)
    const body: any = {
      queries: [
        { datasourceId:this.id,
          orgId: this.id,
          queryText: query.queryText,
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
          let values = []
          for (let key in response.data.results){
            for (let result in response.data.results[key].frames){
              for (let col in response.data.results[key].frames[result].data.values[0]) {
                values.push({text: response.data.results[key].frames[result].data.values[0][col]})
              }
            }
          }
          return values
        }).catch(err =>{
          console.log(err)
          err.isHandled = true; // Avoid extra popup warning
          return ({ text:"ERROR"})
        });
    return backendQuery

  }
}
