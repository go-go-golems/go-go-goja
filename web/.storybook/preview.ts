import type { Preview } from "@storybook/react";
import { initialize, mswLoader } from "msw-storybook-addon";
import "../src/theme/index.css";

initialize();

const preview: Preview = {
  parameters: {
    controls: {
      matchers: {
        color: /(background|color)$/i
      }
    },
    layout: "fullscreen"
  },
  loaders: [mswLoader]
};

export default preview;
