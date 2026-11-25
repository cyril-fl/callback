import type { components } from './api';

type S = components['schemas'];

export type Application = S['domain.Application'];
export type DeletedApplication = S['domain.DeletedApplication'];
export type NewApplication = S['domain.NewApplication'];
export type Changes = S['domain.Changes'];
export type Source = NonNullable<S['domain.NewApplication']['source']>;
export type Status = S['domain.Status'];
export type BatchResult = S['domain.BatchResult'];
export type Token = S['domain.Token'];
export type TokenResult = S['createapitoken.Result'];
export type Backup = S['domain.Backup'];
export type ImportResult = S['importdata.Result'];
export type PurgeResult = S['purgeapplications.Result'];
export type TokenInput = S['createapitoken.Input'];
export type Stats = S['readstats.Stats'];
export type SourceCount = S['listsources.Source'];
export type StatusChange = S['domain.StatusChange'];
export type Me = S['login.MeResult'];

export const STATUSES: Status[] = ['applied', 'responded', 'interview', 'offer', 'rejected'];
// The known sources: the API checks their detail. Any other name is a source too.
export const SOURCES: Source[] = ['spontaneous', 'job_post', 'recruiter'];
export type Preferences = S['domain.Preferences'];
export type PreferenceChanges = S['domain.PreferenceChanges'];
