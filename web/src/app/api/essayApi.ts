import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import type {
  BootstrapResponse,
  EvaluateResponse,
  EvaluationBootstrapResponse,
  ProfileName,
  ProfilesBootstrapResponse,
  SessionSummary
} from "@/features/meet-session/types";

type SessionEnvelope = {
  session: SessionSummary;
};

type EvaluateRequest = {
  sessionId: string;
  source: string;
};

const meetSessionSnapshotPrefix = "/api/essay/sections/meet-a-session/session/";
const profilesSnapshotPrefix = "/api/essay/sections/profiles-change-behavior/session/";
const codeFlowSessionPrefix = "/api/essay/sections/what-happened-to-my-code/session/";

export const essayApi = createApi({
  reducerPath: "essayApi",
  baseQuery: fetchBaseQuery({ baseUrl: "" }),
  tagTypes: ["MeetSession", "ProfileSession", "CodeSession"],
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
  useEvaluateCodeFlowMutation
} = essayApi;
