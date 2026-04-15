import type { Decorator } from "@storybook/react";
import { Provider } from "react-redux";
import { createAppStore, type RootState } from "@/app/store";

export function withEssayProviders(
  preloadedState?: Partial<RootState>
): Decorator {
  return (Story) => {
    const store = createAppStore(preloadedState);
    return (
      <Provider store={store}>
        <Story />
      </Provider>
    );
  };
}
