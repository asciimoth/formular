export type FormularTarget = HTMLElement | string;

export type FormularOutbox = (message: FrontendMessage | UnknownFrontendMessage) => void;

export interface FormularMenuOptions {
  classPrefix?: string;
  prefix?: string;
  defaultTheme?: boolean;
}

export interface MessageBase {
  type: string;
  menuId: string;
  menuGeneration?: number;
  blockGeneration?: number;
}

export interface FieldRef {
  blockId: string;
  fieldId: string;
  elementPath?: ElementPathSegment[];
}

export interface ElementPathSegment {
  arrayFieldId: string;
  elementId: string;
}

export type FieldValue = string | number | boolean | null | ArrayElementValue[];

export interface ArrayElementValue {
  id: string;
  template: string;
  values: Record<string, FieldValue>;
}

export interface Copyable {
  text: string;
}

export type TextFormat = "plain" | "markdown" | "code";
export type FieldKind = "text" | "int" | "float" | "file" | "checkbox" | "radio" | "range" | "array";
export type ValidationStatus = "unset" | "ok" | "warn" | "error";
export type LogLevel = "trace" | "debug" | "info" | "warn" | "error" | "panic";

export interface ItemBase {
  type: string;
  id: string;
  help?: string;
}

export interface HeaderItem extends ItemBase {
  type: "header";
  text: string;
}

export interface LabelItem extends ItemBase {
  type: "label";
  text: string;
  format?: TextFormat;
  syntax?: string;
}

export interface ProgressbarItem extends ItemBase {
  type: "progressbar";
  label: string;
  progress: number;
}

export interface LogLine {
  level: LogLevel;
  text: string;
}

export interface LogsItem extends ItemBase {
  type: "logs";
  label: string;
  logs: LogLine[];
}

export interface ButtonItem extends ItemBase {
  type: "button";
  label: string;
  inactive?: boolean;
}

export interface FieldItem extends ItemBase {
  type: "field";
  kind: FieldKind;
  label: string;
  value?: FieldValue;
  placeholder?: string;
  readonly?: boolean;
  required?: boolean;
  validate?: boolean;
  status?: ValidationStatus;
  statusText?: string;
  secret?: boolean;
  multiline?: boolean;
  subtype?: string;
  autocomplete?: {
    enabled?: boolean;
    tag?: string;
  };
  allowedValues?: FieldValue[];
  min?: number;
  max?: number;
  fraction?: number;
  maxBytes?: number;
  accept?: string[];
  templates?: ArrayTemplate[];
  elements?: ArrayElement[];
  copyable?: Copyable;
}

export type Item = HeaderItem | LabelItem | ProgressbarItem | LogsItem | ButtonItem | FieldItem;

export interface ArrayTemplate {
  name: string;
  label?: string;
  items: Exclude<Item, HeaderItem>[];
}

export interface ArrayElement {
  id: string;
  template: string;
  items: Exclude<Item, HeaderItem>[];
  copyable?: Copyable;
}

export interface Block {
  id: string;
  order: number;
  generation: number;
  form: boolean;
  inactive?: boolean;
  collapsible?: boolean;
  collapsed?: boolean;
  copyable?: Copyable;
  items: Item[];
}

export interface MenuSnapshotMessage extends MessageBase {
  type: "menu.snapshot";
  force?: boolean;
  blocks: Block[];
}

export interface BlockSnapshotMessage extends MessageBase {
  type: "block.snapshot";
  block: Block;
}

export interface BlockDeleteMessage extends MessageBase {
  type: "block.delete";
  blockId: string;
}

export interface FieldStatusMessage extends MessageBase {
  type: "field.status";
  field: FieldRef;
  status: ValidationStatus;
  statusText?: string;
  readonly?: boolean;
}

export interface AutocompleteHintsMessage extends MessageBase {
  type: "autocomplete.hints";
  field: FieldRef;
  prefix: string;
  hints: string[];
}

export type BackendMessage =
  | MenuSnapshotMessage
  | BlockSnapshotMessage
  | BlockDeleteMessage
  | FieldStatusMessage
  | AutocompleteHintsMessage;

export interface FieldUpdateMessage extends MessageBase {
  type: "field.update";
  field: FieldRef;
  value: FieldValue;
}

export interface FieldValidateMessage extends MessageBase {
  type: "field.validate";
  field: FieldRef;
  value: FieldValue;
}

export interface FormApplyMessage extends MessageBase {
  type: "form.apply";
  blockId: string;
  values: Record<string, FieldValue>;
}

export interface ButtonPressMessage extends MessageBase {
  type: "button.press";
  blockId: string;
  elementPath?: ElementPathSegment[];
  buttonId: string;
}

export interface AutocompleteRequestMessage extends MessageBase {
  type: "autocomplete.request";
  field: FieldRef;
  prefix: string;
}

export interface UnknownFrontendMessage extends MessageBase {
  [key: string]: unknown;
}

export type FrontendMessage =
  | FieldUpdateMessage
  | FieldValidateMessage
  | FormApplyMessage
  | ButtonPressMessage
  | AutocompleteRequestMessage;

export declare class FormularMenu {
  constructor(target: FormularTarget, menuId: string, outbox: FormularOutbox, options?: FormularMenuOptions);
  feed(message: BackendMessage | MessageBase | null | undefined): boolean;
  destroy(): void;
}

export default FormularMenu;
