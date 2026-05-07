// PropertyValuation is the payload accepted by the API when an operator
// submits a property valuation for a v2 mortgage application. The value is
// in pounds (whole units; not pence). The API does not apply a default — a
// missing or non-positive value is rejected at the boundary.
export interface PropertyValuation {
  applicationId: string;
  propertyValue: number;
}
