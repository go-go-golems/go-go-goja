import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import type {
  BootstrapResponse,
  EvaluateResponse,
  EvaluationBootstrapResponse,
  EvaluationRecord,
  PersistenceBootstrapResponse,
  ProfileName,
  ProfilesBootstrapResponse,
  SessionExport,
  SessionRecord,
  SessionSummary
  ,
  TimeoutBootstrapResponse
} from "@/features/meet-session/types";

type SessionEnvelope = {
  session: SessionSummary;
};

type EvaluateRequest = {
  sessionId: string;
  source: string;
};

type RawSessionEnvelope = {
  sessions: SessionRecord[];
};

type HistoryEnvelope = {
  history: EvaluationRecord[];
};

const meetSessionSnapshotPrefix = "/api/essay/sections/meet-a-session/session/";
const profilesSnapshotPrefix = "/api/essay/sections/profiles-change-behavior/session/";
const codeFlowSessionPrefix = "/api/essay/sections/what-happened-to-my-code/session/";

export const essayApi = createApi({
  reducerPath: "essayApi",
  baseQuery: fetchBaseQuery({ baseUrl: "" }),
  tagTypes: ["MeetSession", "ProfileSession", "CodeSession", "PersistentSession"],
  endpoints: (builder) => ({
    getMeetSessionBootstrap: builder.query<BootstrapResponse, void>({
      query: () => "/api/essay/sections/meet-a-session"
    }),
    createMeetSession: builder.mutation<SessionSummary, void>({
      query: () => ({
        url: "/api/essay/sections/meet-a-session/session",
        method: "POST"
      }),
      transformResponse: (response: SessionEnvelope) => response.session,
      invalidatesTags: (_result, _error, _arg) => ["MeetSession"]
    }),
    getMeetSessionSnapshot: builder.query<SessionSummary, string>({
      query: (sessionId: string) =>
        `${meetSessionSnapshotPrefix}${encodeURIComponent(sessionId)}`,
      transformResponse: (response: SessionEnvelope) => response.session,
      providesTags: (_result, _error, sessionId) => [{ type: "MeetSession", id: sessionId }]
    }),
    getProfilesBootstrap: builder.query<ProfilesBootstrapResponse, void>({
      query: () => "/api/essay/sections/profiles-change-behavior"
    }),
    createProfileSession: builder.mutation<SessionSummary, ProfileName>({
      query: (profile: ProfileName) => ({
        url: "/api/essay/sections/profiles-change-behavior/session",
        method: "POST",
        body: { profile }
      }),
      transformResponse: (response: SessionEnvelope) => response.session,
      invalidatesTags: (_result, _error, _arg) => ["ProfileSession"]
    }),
    getProfileSessionSnapshot: builder.query<SessionSummary, string>({
      query: (sessionId: string) =>
        `${profilesSnapshotPrefix}${encodeURIComponent(sessionId)}`,
      transformResponse: (response: SessionEnvelope) => response.session,
      providesTags: (_result, _error, sessionId) => [{ type: "ProfileSession", id: sessionId }]
    }),
    getCodeFlowBootstrap: builder.query<EvaluationBootstrapResponse, void>({
      query: () => "/api/essay/sections/what-happened-to-my-code"
    }),
    createCodeFlowSession: builder.mutation<SessionSummary, ProfileName | void>({
      query: (profile) => ({
        url: "/api/essay/sections/what-happened-to-my-code/session",
        method: "POST",
        body: profile ? { profile } : {}
      }),
      transformResponse: (response: SessionEnvelope) => response.session,
      invalidatesTags: (_result, _error, _arg) => ["CodeSession"]
    }),
    evaluateCodeFlow: builder.mutation<EvaluateResponse, EvaluateRequest>({
      query: ({ sessionId, source }) => ({
        url: `${codeFlowSessionPrefix}${encodeURIComponent(sessionId)}/evaluate`,
        method: "POST",
        body: { source }
      }),
      invalidatesTags: (_result, _error, arg) => [{ type: "CodeSession", id: arg.sessionId }]
    }),
    getPersistenceBootstrap: builder.query<PersistenceBootstrapResponse, void>({
      query: () => "/api/essay/sections/persistence-history-and-restore"
    }),
    getTimeoutBootstrap: builder.query<TimeoutBootstrapResponse, void>({
      query: () => "/api/essay/sections/timeouts-are-part-of-the-contract"
    }),
    listPersistentSessions: builder.query<SessionRecord[], void>({
      query: () => "/api/sessions",
      transformResponse: (response: RawSessionEnvelope) => response.sessions,
      providesTags: (result) =>
        result
          ? [
              ...result.map((session) => ({ type: "PersistentSession" as const, id: session.SessionID })),
              { type: "PersistentSession" as const, id: "LIST" }
            ]
          : [{ type: "PersistentSession" as const, id: "LIST" }]
    }),
    createPersistentSession: builder.mutation<SessionSummary, void>({
      query: () => ({
        url: "/api/sessions",
        method: "POST"
      }),
      transformResponse: (response: SessionEnvelope) => response.session,
      invalidatesTags: [{ type: "PersistentSession", id: "LIST" }]
    }),
    evaluatePersistentSession: builder.mutation<EvaluateResponse, EvaluateRequest>({
      query: ({ sessionId, source }) => ({
        url: `/api/sessions/${encodeURIComponent(sessionId)}/evaluate`,
        method: "POST",
        body: { source }
      }),
      invalidatesTags: (_result, _error, arg) => [{ type: "PersistentSession", id: arg.sessionId }]
    }),
    getPersistentSessionHistory: builder.query<EvaluationRecord[], string>({
      query: (sessionId: string) => `/api/sessions/${encodeURIComponent(sessionId)}/history`,
      transformResponse: (response: HistoryEnvelope) => response.history,
      providesTags: (_result, _error, sessionId) => [{ type: "PersistentSession", id: sessionId }]
    }),
    getPersistentSessionExport: builder.query<SessionExport, string>({
      query: (sessionId: string) => `/api/sessions/${encodeURIComponent(sessionId)}/export`,
      providesTags: (_result, _error, sessionId) => [{ type: "PersistentSession", id: sessionId }]
    }),
    deletePersistentSession: builder.mutation<{ deleted: boolean }, string>({
      query: (sessionId: string) => ({
        url: `/api/sessions/${encodeURIComponent(sessionId)}`,
        method: "DELETE"
      }),
      invalidatesTags: (_result, _error, sessionId) => [
        { type: "PersistentSession", id: sessionId },
        { type: "PersistentSession", id: "LIST" }
      ]
    }),
    restorePersistentSession: builder.mutation<SessionSummary, string>({
      query: (sessionId: string) => ({
        url: `/api/sessions/${encodeURIComponent(sessionId)}/restore`,
        method: "POST"
      }),
      transformResponse: (response: SessionEnvelope) => response.session,
      invalidatesTags: (_result, _error, sessionId) => [
        { type: "PersistentSession", id: sessionId },
        { type: "PersistentSession", id: "LIST" }
      ]
    })
  })
});

export const {
  useGetMeetSessionBootstrapQuery,
  useCreateMeetSessionMutation,
  useGetMeetSessionSnapshotQuery,
  useGetProfilesBootstrapQuery,
  useCreateProfileSessionMutation,
  useGetProfileSessionSnapshotQuery,
  useGetCodeFlowBootstrapQuery,
  useCreateCodeFlowSessionMutation,
  useEvaluateCodeFlowMutation,
  useGetPersistenceBootstrapQuery,
  useGetTimeoutBootstrapQuery,
  useListPersistentSessionsQuery,
  useCreatePersistentSessionMutation,
  useEvaluatePersistentSessionMutation,
  useGetPersistentSessionHistoryQuery,
  useGetPersistentSessionExportQuery,
  useDeletePersistentSessionMutation,
  useRestorePersistentSessionMutation
} = essayApi;
