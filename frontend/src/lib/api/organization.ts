import { apiRequest } from "./client";
import type { Department, DepartmentInput, Position, PositionInput } from "./types";

export function listDepartments(): Promise<Department[]> {
  return apiRequest<Department[]>("/api/v1/admin/organization/departments");
}

export function getDepartment(id: string): Promise<Department> {
  return apiRequest<Department>(`/api/v1/admin/organization/departments/${encodeURIComponent(id)}`);
}

export function createDepartment(input: DepartmentInput): Promise<Department> {
  return apiRequest<Department>("/api/v1/admin/organization/departments", {
    method: "POST",
    body: input,
  });
}

export function updateDepartment(id: string, input: DepartmentInput): Promise<Department> {
  return apiRequest<Department>(
    `/api/v1/admin/organization/departments/${encodeURIComponent(id)}`,
    { method: "PUT", body: input },
  );
}

export function deleteDepartment(id: string): Promise<void> {
  return apiRequest<void>(`/api/v1/admin/organization/departments/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}

export function listPositions(): Promise<Position[]> {
  return apiRequest<Position[]>("/api/v1/admin/organization/positions");
}

export function createPosition(input: PositionInput): Promise<Position> {
  return apiRequest<Position>("/api/v1/admin/organization/positions", {
    method: "POST",
    body: input,
  });
}

export function updatePosition(id: string, input: PositionInput): Promise<Position> {
  return apiRequest<Position>(`/api/v1/admin/organization/positions/${encodeURIComponent(id)}`, {
    method: "PUT",
    body: input,
  });
}

export function deletePosition(id: string): Promise<void> {
  return apiRequest<void>(`/api/v1/admin/organization/positions/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}
