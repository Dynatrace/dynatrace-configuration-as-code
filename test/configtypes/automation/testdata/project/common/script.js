// optional import of sdk modules
import { metadataClient } from '@dynatrace-sdk/client-metadata';
import { executionsClient } from '@dynatrace-sdk/client-automation';

export default async function ({ execution_id }) {
  // your code goes here
  const me = await metadataClient.getUserInfo();
  console.log('Automated script execution on behalf of', me.userName);

  console.log({{`{{`}} event(){{`}}`}})
  // get the current execution
  const ex = await executionsClient.getExecution({ id: execution_id });

  return { ...me, triggeredBy: ex.trigger };
}
