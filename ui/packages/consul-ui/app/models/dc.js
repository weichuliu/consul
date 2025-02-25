import Model, { attr } from '@ember-data/model';

export const PRIMARY_KEY = 'uid';
export const FOREIGN_KEY = 'Datacenter';
export const SLUG_KEY = 'Name';

export default class Datacenter extends Model {
  @attr('string') uri;
  @attr('string') Name;
  // autopilot/state
  @attr('boolean') Healthy;
  @attr('number') FailureTolerance;
  @attr('number') OptimisticFailureTolerance;
  @attr('string') Leader;
  @attr() Voters; // []
  @attr() Servers; // [] the API uses {} but we reshape that on the frontend
  //
  @attr('boolean') Local;
  @attr('boolean') Primary;
  @attr('string') DefaultACLPolicy;

  @attr('boolean', { defaultValue: () => true }) MeshEnabled;
}
