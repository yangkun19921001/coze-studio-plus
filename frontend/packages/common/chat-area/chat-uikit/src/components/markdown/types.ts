/*
 * Copyright 2025 coze-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import type React from 'react';

import { type ImageOptions } from '@coze-arch/bot-md-box-adapter';

export interface CozeMarkdownProps {
  /** Markdown content to render */
  children: string;
  /** Whether content is being streamed */
  isStreaming?: boolean;
  /** Show streaming indicator (cursor/dots) */
  showIndicator?: boolean;
  /** Image configuration */
  imageOptions?: ImageOptions;
  /** Event callbacks for interactive elements */
  eventCallbacks?: {
    onImageClick?: (e: React.MouseEvent, data: { src: string }) => void;
    onLinkClick?: (
      e: React.MouseEvent,
      data: { url: string; parsedUrl?: URL },
    ) => void;
  };
  /** Custom className */
  className?: string;
  /** Legacy mdBoxProps for compatibility */
  mdBoxProps?: {
    slots?: Record<string, React.ComponentType<unknown>>;
  };
}

export interface StreamingIndicatorProps {
  /** Custom className */
  className?: string;
}
