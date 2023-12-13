export type Nullable<T> = T | null;

export type Undefinable<T> = T | undefined;

export type Maybe<T> = T | undefined | null;

export interface PaginationRequest {
  offset: number;
  limit: number;
}

export interface PaginationResponse<T> {
  data: T[];
  pagination: PaginationRequest & {
    total: number;
  };
}
