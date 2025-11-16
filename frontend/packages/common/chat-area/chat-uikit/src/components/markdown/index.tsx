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

import ReactMarkdown from 'react-markdown';
import { type Components } from 'react-markdown';
import { type FC, useMemo, type ErrorInfo, Component } from 'react';

import remarkGfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';
import classNames from 'classnames';

import { CozeLink } from '../md-box-slots/link';
import { CozeImage } from '../md-box-slots/coze-image';
import { type CozeMarkdownProps } from './types';
import { Table, TableHead, TableCell } from './components/table';
import { StreamingIndicator } from './components/streaming-indicator';
import { CodeBlock } from './components/code-block';

import './index.less';

/**
 * Error boundary for ReactMarkdown to catch parsing errors
 */
class MarkdownErrorBoundary extends Component<
  { children: React.ReactNode; fallback?: React.ReactNode; plainText: string },
  { hasError: boolean; error?: Error }
> {
  constructor(props: {
    children: React.ReactNode;
    fallback?: React.ReactNode;
    plainText: string;
  }) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error) {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('ReactMarkdown parsing error:', error, errorInfo);
    console.warn('Fallback to plain text display');
  }

  render() {
    if (this.state.hasError) {
      // Fallback: display as plain text with basic formatting
      return (
        this.props.fallback || (
          <div
            style={{
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-word',
              fontFamily: 'inherit',
              lineHeight: '1.6',
            }}
          >
            {this.props.plainText}
          </div>
        )
      );
    }

    return this.props.children;
  }
}

/**
 * Coze React Markdown Component
 *
 * A wrapper around react-markdown with Coze-specific customizations:
 * - GitHub Flavored Markdown support (tables, strikethrough, etc.)
 * - Syntax highlighting for code blocks
 * - Custom image and link components with event callbacks
 * - Streaming indicator for real-time rendering
 * - Compatible with existing Calypso-based components
 */
export const CozeReactMarkdown: FC<CozeMarkdownProps> = ({
  children,
  isStreaming = false,
  showIndicator = false,
  imageOptions,
  eventCallbacks,
  className,
  mdBoxProps,
}) => {
  // Custom components for markdown elements
  // Note: We use minimal customization to avoid conflicts with remark-gfm
  const components: Components = useMemo(
    () => ({
      // Image component with custom click handling
      img: props => (
        <CozeImage
          {...props}
          src={props.src || ''}
          imageOptions={imageOptions}
          onImageClick={(e, data) => {
            // Ensure src is not null before calling callback
            if (data.src) {
              eventCallbacks?.onImageClick?.(e, data as { src: string });
            }
          }}
        />
      ),

      // Link component with custom click handling
      a: ({ children: linkChildren, href, ...props }) => (
        <CozeLink
          href={href || ''}
          onLinkClick={(e, data) => {
            // Pass through the complete data including exts and openLink
            eventCallbacks?.onLinkClick?.(e, data);
          }}
        >
          {linkChildren}
        </CozeLink>
      ),

      // Table components with responsive wrapper
      table: props => <Table {...props} />,
      th: props => <TableHead {...props} />,
      td: props => <TableCell {...props} />,

      // Code block with language label in top-left corner
      // 参考 kaflow-web 的实现方式
      code: ({ inline, className, children, ...props }: any) => {
        const match = /language-(\w+)/.exec(className || '');
        const language = match ? match[1] : '';
        
        // 如果是代码块（有语言标识且不是内联）
        if (!inline && language) {
          // 如果是 JSON 代码块，格式化并处理 \n 转义字符
          let formattedChildren = children;
          if (language === 'json' || language === 'jsonc') {
            try {
              const codeString = String(children).replace(/\n$/, '');
              // 解析 JSON
              const parsed = JSON.parse(codeString);
              
              // 格式化 JSON，使用 2 空格缩进
              formattedChildren = JSON.stringify(parsed, null, 2);
              
              // 处理 JSON 字符串值中的 \n 转义字符
              // 使用更精确的正则表达式匹配 JSON 字符串值（值部分，不是键部分）
              formattedChildren = formattedChildren.replace(
                /(:\s*")((?:[^"\\]|\\.)*)(")/g,
                (match, prefix, content, suffix) => {
                  // 如果包含 \n 转义字符，替换为实际换行
                  if (content.includes('\\n')) {
                    // 将 \\n 替换为实际换行符 \n
                    // 注意：需要正确处理其他转义字符
                    const processed = content
                      .replace(/\\\\/g, '\u0001') // 临时标记双反斜杠
                      .replace(/\\n/g, '\n') // 将 \n 转换为实际换行
                      .replace(/\u0001/g, '\\\\'); // 恢复双反斜杠
                    return prefix + processed + suffix;
                  }
                  return match;
                }
              );
            } catch (e) {
              // 如果解析失败，尝试直接处理 \n 转义字符
              formattedChildren = String(children).replace(/\\n/g, '\n');
            }
          }
          
          return (
            <CodeBlock inline={false} className={className} {...props}>
              {formattedChildren}
            </CodeBlock>
          );
        }
        
        // 内联代码：直接返回 code 标签，添加 inline-code 类名
        return (
          <code className={`coze-markdown-code-inline ${className || ''}`} {...props}>
            {children}
          </code>
        );
      },
      
      // 处理段落中的 \n 转义字符
      p: ({ children, ...props }: any) => {
        // 如果 children 是字符串且包含 \n，需要处理换行
        const processChildren = (node: any): any => {
          if (typeof node === 'string') {
            // 将 \n 转换为 <br /> 或使用 white-space: pre-wrap
            return node;
          }
          if (Array.isArray(node)) {
            return node.map(processChildren);
          }
          return node;
        };
        
        return (
          <p style={{ whiteSpace: 'pre-wrap' }} {...props}>
            {processChildren(children)}
          </p>
        );
      },

      // IMPORTANT: Do NOT spread mdBoxProps.slots here!
      // mdBoxProps.slots contains Calypso-specific components that are incompatible
      // with react-markdown's Components API and will break remark-gfm's inTable context.
      // If custom slots are needed in the future, they must be explicitly mapped
      // and validated for react-markdown compatibility.
    }),
    [imageOptions, eventCallbacks],
  );

  // Process and clean content
  const processedContent = children;

  return (
    <div
      className={classNames('coze-react-markdown', className, {
        'coze-react-markdown-streaming': isStreaming,
      })}
    >
      <MarkdownErrorBoundary plainText={processedContent}>
        <ReactMarkdown
          remarkPlugins={[remarkGfm]}
          rehypePlugins={[rehypeRaw]}
          components={components}
        >
          {processedContent}
        </ReactMarkdown>
      </MarkdownErrorBoundary>

      {/* Streaming indicator */}
      {isStreaming && showIndicator ? <StreamingIndicator /> : null}
    </div>
  );
};

CozeReactMarkdown.displayName = 'CozeReactMarkdown';
