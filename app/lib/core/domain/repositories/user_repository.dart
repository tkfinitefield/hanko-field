import 'package:app/core/data/dtos/user_dto.dart';
import 'package:app/core/domain/entities/user.dart';

/// DTO ⇔ ドメイン変換をリポジトリ実装で共有するための mixin
mixin UserDtoMapper {
  UserProfile mapUserProfile(UserProfileDto dto) => dto.toDomain();
  UserProfileDto mapUserProfileToDto(UserProfile domain) =>
      UserProfileDto.fromDomain(domain);

  UserAddress mapUserAddress(UserAddressDto dto) => dto.toDomain();
  UserAddressDto mapUserAddressToDto(UserAddress domain) =>
      UserAddressDto.fromDomain(domain);

  UserPaymentMethod mapPaymentMethod(UserPaymentMethodDto dto) =>
      dto.toDomain();
  UserPaymentMethodDto mapPaymentMethodToDto(UserPaymentMethod domain) =>
      UserPaymentMethodDto.fromDomain(domain);

  UserFavoriteDesign mapFavoriteDesign(UserFavoriteDesignDto dto) =>
      dto.toDomain();
  UserFavoriteDesignDto mapFavoriteDesignToDto(UserFavoriteDesign domain) =>
      UserFavoriteDesignDto.fromDomain(domain);
}

abstract class UserRepository with UserDtoMapper {
  Future<UserProfile> fetchCurrentUser();
  Future<UserProfile> updateProfile(UserProfile profile);

  Future<List<UserAddress>> fetchAddresses();
  Future<UserAddress> upsertAddress(UserAddress address);
  Future<void> deleteAddress(String addressId);

  Future<List<UserPaymentMethod>> fetchPaymentMethods();
  Future<void> removePaymentMethod(String methodId);

  Future<List<UserFavoriteDesign>> fetchFavorites();
  Future<void> addFavorite(UserFavoriteDesign favorite);
  Future<void> removeFavorite(String favoriteId);
}
