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

import { type MouseEvent, type FC, type ComponentType, useRef } from 'react';

import {
  type IOnImageClickParams,
  type IOnLinkClickParams,
  type IBaseContentProps,
  type MdBoxProps,
} from '@coze-common/chat-uikit-shared';
import { Image } from '@coze-arch/bot-md-box-adapter/slots';
import { type ImageOptions } from '@coze-arch/bot-md-box-adapter';

import { CozeLink } from '../../md-box-slots/link';
import { CozeImage } from '../../md-box-slots/coze-image';
import { CozeReactMarkdown } from '../../markdown';
import { LazyCozeMdBox } from '../../common/coze-md-box/lazy';
import { isText } from '../../../utils/is-text';
import './index.less';

// Feature flag: Use react-markdown instead of Calypso
// Set to true to use the new react-markdown renderer
const USE_REACT_MARKDOWN = true;

const buildLinkExtra = (eventData: {
  url?: string;
  parsedUrl?: URL;
  exts?: Record<string, unknown>;
}) => {
  const base =
    typeof window !== 'undefined' && window.location?.origin
      ? window.location.origin
      : 'http://localhost';

  let parsed = eventData.parsedUrl;
  if (!parsed) {
    try {
      parsed = new URL(eventData.url ?? '', base);
    } catch {
      parsed = new URL(base);
    }
  }

  return {
    url: eventData.url ?? parsed.href,
    parsedUrl: parsed,
    exts: eventData.exts ?? {},
  };
};

export type IMessageContentProps = IBaseContentProps & {
  onImageClick?: (params: IOnImageClickParams) => void;
  mdBoxProps?: MdBoxProps;
  enableAutoSizeImage?: boolean;
  imageOptions?: ImageOptions;
  onLinkClick?: (
    params: IOnLinkClickParams,
    event: MouseEvent<Element, globalThis.MouseEvent>,
  ) => void;
};

export const TextContent: FC<IMessageContentProps> = props => {
  const {
    message,
    readonly,
    onImageClick,
    onLinkClick,
    mdBoxProps,
    enableAutoSizeImage,
    imageOptions,
  } = props;
  const contentRef = useRef<HTMLDivElement | null>(null);
  const { content } = message;

  if (!isText(content)) {
    return null;
  }

  const isStreaming = !message.is_finish;
  const text = content.slice(0, message.broken_pos ?? Infinity);

  const reactMarkdownMdBoxProps = mdBoxProps?.slots
    ? {
        slots: mdBoxProps.slots as unknown as Record<
          string,
          ComponentType<unknown>
        >,
      }
    : undefined;

  // Use react-markdown if feature flag is enabled
  if (USE_REACT_MARKDOWN) {
    return (
      <div
        className="chat-uikit-text-content"
        data-testid="bot.ide.chat_area.message.text-answer-message-content"
        ref={contentRef}
        data-grab-mark={message.message_id}
        data-grab-source={message.source}
      >
        <CozeReactMarkdown
          isStreaming={isStreaming}
          showIndicator={isStreaming}
          imageOptions={{ forceHttps: !IS_OPEN_SOURCE, ...imageOptions }}
          eventCallbacks={{
            onImageClick: (e, eventData) => {
              eventData.src &&
                onImageClick?.({
                  message,
                  extra: { url: eventData.src },
                });
            },
            onLinkClick: (e, eventData) => {
              onLinkClick?.(
                {
                  message,
                  extra: buildLinkExtra(eventData),
                },
                e,
              );

              if (readonly) {
                e.preventDefault();
                e.stopPropagation();
              }
            },
          }}
          mdBoxProps={reactMarkdownMdBoxProps}
        >
          {text}
        </CozeReactMarkdown>
      </div>
    );
  }

  // Fallback to Calypso renderer
  const MdBoxLazy = LazyCozeMdBox;
  return (
    <div
      className="chat-uikit-text-content"
      data-testid="bot.ide.chat_area.message.text-answer-message-content"
      ref={contentRef}
      data-grab-mark={message.message_id}
      data-grab-source={message.source}
    >
      <MdBoxLazy
        markDown={text}
        autoFixSyntax={{ autoFixEnding: isStreaming }}
        showIndicator={isStreaming}
        smooth={isStreaming}
        imageOptions={{ forceHttps: !IS_OPEN_SOURCE, ...imageOptions }}
        eventCallbacks={{
          onImageClick: (e, eventData) => {
            eventData.src &&
              onImageClick?.({
                message,
                extra: { url: eventData.src },
              });
          },
          onLinkClick: (e, eventData) => {
            onLinkClick?.(
              {
                message,
                extra: { ...eventData },
              },
              e,
            );

            if (readonly) {
              e.preventDefault();
              e.stopPropagation();
            }
          },
        }}
        {...mdBoxProps}
        slots={{
          Image: enableAutoSizeImage ? CozeImage : Image,
          Link: CozeLink,
          ...mdBoxProps?.slots,
        }}
      ></MdBoxLazy>
    </div>
  );
};

TextContent.displayName = 'TextContent';
