import { Provider } from "react-redux";
import { MeetSessionPage } from "@/features/meet-session/MeetSessionPage";
import { createAppStore } from "@/app/store";

const store = createAppStore();

export function EssayApp() {
  return (
    <Provider store={store}>
      <MeetSessionPage />
    </Provider>
  );
}
