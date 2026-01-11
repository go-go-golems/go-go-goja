declare module "fs" {
  export function readFileSync(path: string): string;
  export function writeFileSync(path: string, data: string): void;
}

declare module "exec" {
  export function run(cmd: string, args: string[]): string;
}

declare module "database" {
  export function configure(driver: string, dsn: string): void;
  export function query(sql: string, args?: unknown[]): unknown;
  export function exec(sql: string, args?: unknown[]): void;
  export function close(): void;
}

declare module "*.svg" {
  const content: string;
  export default content;
}
