import { createSlice, type PayloadAction } from "@reduxjs/toolkit";

export type MeetSessionState = {
  activeSessionId: string | null;
};

const initialState: MeetSessionState = {
  activeSessionId: null
};

const meetSessionSlice = createSlice({
  name: "meetSession",
  initialState,
  reducers: {
    setActiveSessionId(state, action: PayloadAction<string>) {
      state.activeSessionId = action.payload;
    },
    clearActiveSession(state) {
      state.activeSessionId = null;
    }
  }
});

export const { setActiveSessionId, clearActiveSession } = meetSessionSlice.actions;
export const meetSessionReducer = meetSessionSlice.reducer;
