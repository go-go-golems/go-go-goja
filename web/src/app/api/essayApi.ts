import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import type { BootstrapResponse, SessionSummary } from "@/features/meet-session/types";

type SessionEnvelope = {
  session: SessionSummary;
};

const meetSessionSnapshotPrefix = "/api/essay/sections/meet-a-session/session/";

export const essayApi = createApi({
  reducerPath: "essayApi",
  baseQuery: fetchBaseQuery({ baseUrl: "" }),
  tagTypes: ["MeetSession"],
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
    })
  })
});

export const {
  useGetMeetSessionBootstrapQuery,
  useCreateMeetSessionMutation,
  useGetMeetSessionSnapshotQuery
} = essayApi;
