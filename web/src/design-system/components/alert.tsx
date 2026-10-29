import { Alert as SharedAlert } from "@want-foundation/desyst-react";
import type { ReactNode } from "react";
import {
  CircleCheckIcon,
  InfoIcon,
  TriangleAlertIcon,
  OctagonXIcon,
} from "lucide-react";
import type { FeedbackTone } from "./feedback-tone";

const icons = {
  neutral: InfoIcon,
  success: CircleCheckIcon,
  warning: TriangleAlertIcon,
  error: OctagonXIcon,
};
export function Alert({
  title,
  children,
  tone = "neutral",
  action,
  announce = false,
}: {
  title: string;
  children: ReactNode;
  tone?: FeedbackTone;
  action?: ReactNode;
  announce?: boolean;
}) {
  const Icon = icons[tone];
  return (
    <SharedAlert
      className="wk-alert"
      data-tone={tone}
      role={announce ? "status" : undefined}
    >
      <Icon aria-hidden="true" />
      <div>
        <p className="wk-alert-title">{title}</p>
        <div className="wk-description">{children}</div>
        {action && <div className="wk-alert-action">{action}</div>}
      </div>
    </SharedAlert>
  );
}
