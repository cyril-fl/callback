import type { components } from './api';

type S = components['schemas'];

export type Application = S['domain.Application'];
export type NewApplication = S['domain.NewApplication'];
export type Changes = S['domain.Changes'];
export type Source = S['domain.Source'];
export type Status = S['domain.Status'];
export type BatchResult = S['domain.BatchResult'];
export type Token = S['domain.Token'];
export type TokenResult = S['createapitoken.Result'];
export type TokenInput = S['createapitoken.Input'];
export type Stats = S['readstats.Stats'];
export type Me = S['login.MeResult'];

export const STATUSES: Status[] = ['applied', 'responded', 'interview', 'offer', 'rejected'];
export const SOURCES: Source[] = ['spontaneous', 'job_post', 'recruiter'];
