export type ProviderConfig = {
  client_id: string;
  client_secret: string;
  authorize_url?: string;
  token_url?: string;
  scopes?: string[];
  auth_params?: Record<string, string>;
};

export type ProvidersConfig = Record<string, ProviderConfig>;

export type BrokerSession = {
  auth_url: string;
  session_id: string;
};

export type BrokerToken = {
  token_json: string;
  account_label?: string;
  client_id?: string;
  client_secret?: string;
};

export type SessionRecord = {
  provider: string;
  sratCallbackUrl: string;
  createdAt: number;
  // Set after provider callback completes token exchange
  tokenJson?: string;
  accountLabel?: string;
  clientId?: string;
  clientSecret?: string;
  completedAt?: number;
};

export type HealthResponse = {
  status: "ok";
  providers: string[];
};
