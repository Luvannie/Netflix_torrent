import type { ApiErrorDetail, ApiErrorResponse } from "./types";

export class ApiClientError extends Error {
  readonly status: number;
  readonly code: string;
  readonly details: ApiErrorDetail[];
  readonly requestId?: string;

  constructor(args: {
    status: number;
    code: string;
    message: string;
    details?: ApiErrorDetail[];
    requestId?: string;
  }) {
    super(args.message);
    this.name = "ApiClientError";
    this.status = args.status;
    this.code = args.code;
    this.details = args.details ?? [];
    this.requestId = args.requestId;
  }

  static fromResponse(response: ApiErrorResponse, status: number) {
    return new ApiClientError({
      status,
      code: response.error.code,
      message: response.error.message,
      details: response.error.details,
      requestId: response.meta.requestId,
    });
  }
}

