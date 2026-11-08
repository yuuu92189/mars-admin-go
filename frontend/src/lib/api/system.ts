import { apiRequest } from "./client";
import type {
  DictionaryItem,
  DictionaryItemInput,
  DictionaryType,
  DictionaryTypeInput,
  SettingGroup,
} from "./types";

export function listDictionaries(): Promise<DictionaryType[]> {
  return apiRequest<DictionaryType[]>("/api/v1/admin/dictionaries");
}

export function createDictionary(input: DictionaryTypeInput): Promise<DictionaryType> {
  return apiRequest<DictionaryType>("/api/v1/admin/dictionaries", {
    method: "POST",
    body: input,
  });
}

export function updateDictionary(id: string, input: DictionaryTypeInput): Promise<DictionaryType> {
  return apiRequest<DictionaryType>(`/api/v1/admin/dictionaries/${encodeURIComponent(id)}`, {
    method: "PUT",
    body: input,
  });
}

export function deleteDictionary(id: string): Promise<void> {
  return apiRequest<void>(`/api/v1/admin/dictionaries/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}

export function listDictionaryItems(typeId: string): Promise<DictionaryItem[]> {
  return apiRequest<DictionaryItem[]>(
    `/api/v1/admin/dictionaries/${encodeURIComponent(typeId)}/items`,
  );
}

export function createDictionaryItem(input: DictionaryItemInput): Promise<DictionaryItem> {
  return apiRequest<DictionaryItem>("/api/v1/admin/dictionaries/items", {
    method: "POST",
    body: input,
  });
}

export function updateDictionaryItem(
  id: string,
  input: DictionaryItemInput,
): Promise<DictionaryItem> {
  return apiRequest<DictionaryItem>(`/api/v1/admin/dictionaries/items/${encodeURIComponent(id)}`, {
    method: "PUT",
    body: input,
  });
}

export function deleteDictionaryItem(id: string): Promise<void> {
  return apiRequest<void>(`/api/v1/admin/dictionaries/items/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}

export function listSettingGroups(): Promise<SettingGroup[]> {
  return apiRequest<SettingGroup[]>("/api/v1/admin/settings");
}

export function getSettingGroup(code: string): Promise<SettingGroup> {
  return apiRequest<SettingGroup>(`/api/v1/admin/settings/${encodeURIComponent(code)}`);
}

export function updateSettingGroup(
  code: string,
  values: Record<string, unknown>,
): Promise<SettingGroup> {
  return apiRequest<SettingGroup>(`/api/v1/admin/settings/${encodeURIComponent(code)}`, {
    method: "PUT",
    body: values,
  });
}
