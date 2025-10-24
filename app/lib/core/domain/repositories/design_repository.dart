import 'package:app/core/data/dtos/design_dto.dart';
import 'package:app/core/domain/entities/design.dart';

mixin DesignDtoMapper {
  Design mapDesign(DesignDto dto) => dto.toDomain();
  DesignDto mapDesignToDto(Design domain) => DesignDto.fromDomain(domain);

  DesignInput mapDesignInput(DesignInputDto dto) => dto.toDomain();
  DesignInputDto mapDesignInputToDto(DesignInput domain) =>
      DesignInputDto.fromDomain(domain);
}

abstract class DesignRepository with DesignDtoMapper {
  Future<List<Design>> fetchDesigns({int? pageSize, String? pageToken});
  Future<Design> fetchDesign(String designId);
  Future<Design> createDesign(Design design);
  Future<Design> updateDesign(Design design);
  Future<void> deleteDesign(String designId);

  Future<List<Design>> fetchVersions(String designId);
  Future<Design> duplicateDesign(String designId);
  Future<void> requestAiSuggestions(
    String designId,
    Map<String, dynamic> payload,
  );
}
