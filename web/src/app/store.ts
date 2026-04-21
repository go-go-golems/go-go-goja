import {
  combineReducers,
  configureStore
} from "@reduxjs/toolkit";
import { essayApi } from "@/app/api/essayApi";
import { meetSessionReducer } from "@/features/meet-session/meetSessionSlice";

const rootReducer = combineReducers({
  [essayApi.reducerPath]: essayApi.reducer,
  meetSession: meetSessionReducer
});

export type RootState = ReturnType<typeof rootReducer>;

export const createAppStore = (preloadedState?: Partial<RootState>) =>
  configureStore({
    reducer: rootReducer,
    middleware: (getDefaultMiddleware) =>
      getDefaultMiddleware().concat(essayApi.middleware),
    preloadedState: preloadedState as RootState | undefined
  });

export type AppStore = ReturnType<typeof createAppStore>;
export type AppDispatch = AppStore["dispatch"];
