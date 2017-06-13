import React from "react";
import _ from "lodash";

import * as protos from "src/js/protos";

export interface DebugFailureTableProps {
  failures: protos.cockroach.server.serverpb.DebugFailure$Properties[];
}

export class DebugFailureTable extends React.Component<DebugFailureTableProps, {}> {
  render() {
    if (_.isEmpty(this.props.failures)) {
      return null;
    }
    return (
      <div>
        <h2>Failures</h2>
        <table className="failure-table">
          <thead>
            <tr className="failure-table__row failure-table__row--header">
              <td className="failure-table__cell failure-table__cell--header failure-table__cell--short">Node</td>
              <td className="failure-table__cell failure-table__cell--short">Error</td>
            </tr>
          </thead>
          <tbody>
            {
              _.map(this.props.failures, (failure) => (
                <tr className="failure-table__row" key={failure.node_id}>
                  <td className="failure-table__cell failure-table__cell--short">n{failure.node_id}</td>
                  <td className="failure-table__cell">title={failure.error_message}>{failure.error_message}</td>
                </tr>
              ))
            }
          </tbody>
        </table>
      </div>
    );
  }
}
